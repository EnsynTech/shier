package imap

/* TODO:
*		 - Have a counter per config for number of consecutive login errors and backoff (e.g if supplied creds are incorrect)
*		 - Have a DB field "last_login_error" if last login failed
*		 - DB counter for non-campaign emails that the admin should investigate
*		 - Add field to User for numner of non-campaign emails reported
 */
import (
	"bytes"
	"context"
	"mime"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jordan-wright/email"

	"github.com/gophish/gophish/models"
)

// Pattern for GoPhish emails e.g ?rid=AbC1234
// Also accepts quoted-printable 3D and one or two layers of URL encoding,
// which is what some mail gateways produce.
var goPhishRegex = regexp.MustCompile(`(?i)(?:\?|%3F|%253F)rid(?:=|%3D|%253D)(?:3D)?([A-Za-z0-9]{7})`)

const (
	reportSourceForward    = "forward"
	reportSourceComplaint  = "complaint_reply"
	reportSourceAttachment = "attachment"
)

// Monitor is a worker that monitors IMAP servers for reported campaign emails
type Monitor struct {
	cancel func()
}

// Monitor.start() checks for campaign emails
// As each account can have its own polling frequency set we need to run one Go routine for
// each, as well as keeping an eye on newly created user accounts.
func (im *Monitor) start(ctx context.Context) {
	usermap := make(map[int64]int) // Keep track of running go routines, one per user. We assume incrementing non-repeating UIDs (for the case where users are deleted and re-added).

	for {
		select {
		case <-ctx.Done():
			return
		default:
			dbusers, err := models.GetUsers() //Slice of all user ids. Each user gets their own IMAP monitor routine.
			if err != nil {
				log.Error(err)
				break
			}
			for _, dbuser := range dbusers {
				if _, ok := usermap[dbuser.Id]; !ok { // If we don't currently have a running Go routine for this user, start one.
					log.Info("Starting new IMAP monitor for user ", dbuser.Username)
					usermap[dbuser.Id] = 1
					go monitor(dbuser.Id, ctx)
				}
			}
			time.Sleep(10 * time.Second) // Every ten seconds we check if a new user has been created
		}
	}
}

// monitor will continuously login to the IMAP settings associated to the supplied user id (if the user account has IMAP settings, and they're enabled.)
// It also verifies the user account exists, and returns if not (for the case of a user being deleted).
func monitor(uid int64, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 1. Check if user exists, if not, return.
			_, err := models.GetUser(uid)
			if err != nil { // Not sure if there's a better way to determine user existence via id.
				log.Info("User ", uid, " seems to have been deleted. Stopping IMAP monitor for this user.")
				return
			}
			// 2. Check if user has IMAP settings.
			imapSettings, err := models.GetIMAP(uid)
			if err != nil {
				log.Error(err)
				break
			}
			if len(imapSettings) > 0 {
				im := imapSettings[0]
				// 3. Check if IMAP is enabled
				if im.Enabled {
					log.Debug("Checking IMAP for user ", uid, ": ", im.Username, " -> ", im.Host)
					checkForNewEmails(im)
					time.Sleep((time.Duration(im.IMAPFreq) - 10) * time.Second) // Subtract 10 to compensate for the default sleep of 10 at the bottom
				}
			}
		}
		time.Sleep(10 * time.Second)
	}
}

// NewMonitor returns a new instance of imap.Monitor
func NewMonitor() *Monitor {
	im := &Monitor{}
	return im
}

// Start launches the IMAP campaign monitor
func (im *Monitor) Start() error {
	log.Info("Starting IMAP monitor manager")
	ctx, cancel := context.WithCancel(context.Background()) // ctx is the derivedContext
	im.cancel = cancel
	go im.start(ctx)
	return nil
}

// Shutdown attempts to gracefully shutdown the IMAP monitor.
func (im *Monitor) Shutdown() error {
	log.Info("Shutting down IMAP monitor manager")
	im.cancel()
	return nil
}

// checkForNewEmails logs into an IMAP account and checks unread emails for the
// rid campaign identifier.
func checkForNewEmails(im models.IMAP) {
	im.Host = im.Host + ":" + strconv.Itoa(int(im.Port)) // Append port
	mailServer := Mailbox{
		Host:             im.Host,
		TLS:              im.TLS,
		IgnoreCertErrors: im.IgnoreCertErrors,
		User:             im.Username,
		Pwd:              im.Password,
		Folder:           im.Folder,
	}

	msgs, err := mailServer.GetUnread(false, false)
	if err != nil {
		log.Error(err)
		checkSenderReplyMailboxes(im)
		return
	}
	// Update last_succesful_login here via im.Host
	err = models.SuccessfulLogin(&im)

	// The configured mailbox receives forwards. Replies to the phishing
	// From address are read from each sending profile mailbox.
	checkSenderReplyMailboxes(im)

	senders := sendingAddresses(im.UserId)
	var readUIDs []uint32
	var deleteUIDs []uint32
	for _, m := range msgs {
		action := processReportMailboxMessage(im, m, senders)
		if m.UID == 0 || action == mailboxLeave {
			continue
		}
		if action == mailboxDelete {
			deleteUIDs = append(deleteUIDs, m.UID)
			continue
		}
		readUIDs = append(readUIDs, m.UID)
	}
	if len(readUIDs) > 0 {
		if err := mailServer.MarkAsReadUIDs(readUIDs); err != nil {
			log.Error("Unable to mark emails as read: ", err.Error())
		}
	}
	if len(deleteUIDs) > 0 {
		if err := mailServer.DeleteByUID(deleteUIDs); err != nil {
			log.Error("Failed to delete emails: ", err.Error())
		}
	}
}

// checkSenderReplyMailboxes reads each sending profile inbox on the same
// IMAP server. Ordinary replies and complaint replies are recorded there.
// Forwards stay in the report mailbox.
func checkSenderReplyMailboxes(im models.IMAP) {
	smtps, err := models.GetSMTPs(im.UserId)
	if err != nil {
		log.Error(err)
		return
	}
	senders := sendingAddresses(im.UserId)
	for _, smtp := range smtps {
		user := strings.TrimSpace(smtp.Username)
		if user == "" || smtp.Password == "" || !strings.Contains(user, "@") {
			continue
		}
		if strings.EqualFold(user, strings.TrimSpace(im.Username)) {
			continue
		}
		sender := Mailbox{
			Host:             im.Host,
			TLS:              im.TLS,
			IgnoreCertErrors: im.IgnoreCertErrors,
			User:             user,
			Pwd:              smtp.Password,
			Folder:           im.Folder,
		}
		msgs, err := sender.GetUnread(false, false)
		if err != nil {
			log.Error(err)
			continue
		}
		var readUIDs []uint32
		for _, m := range msgs {
			if processSenderMailboxMessage(im, m, senders) && m.UID != 0 {
				readUIDs = append(readUIDs, m.UID)
			}
		}
		if len(readUIDs) == 0 {
			continue
		}
		if err := sender.MarkAsReadUIDs(readUIDs); err != nil {
			log.Error("Unable to mark sender replies as read: ", err.Error())
		}
	}
}

type mailboxAction int

const (
	mailboxLeave mailboxAction = iota
	mailboxRead
	mailboxDelete
)

func processReportMailboxMessage(im models.IMAP, m Email, senders map[string]bool) mailboxAction {
	if im.RestrictDomain != "" {
		domain := senderDomain(m.Email.From)
		if !strings.EqualFold(domain, im.RestrictDomain) {
			log.Debug("Ignoring email as not from company domain: ", domain)
			return mailboxLeave
		}
	}
	if isCampaignOriginal(m.Email, senders) {
		log.Debugf("Ignoring campaign original in report mailbox: %s", m.Email.Subject)
		return mailboxLeave
	}
	forward := isForwardSubject(m.Email.Subject)
	replied := false
	if !forward {
		replied = handleReplyEmail(m.Email)
	}
	rids, err := matchEmail(m.Email)
	if err != nil {
		log.Errorf("Error searching email for rids from user '%s': %s", m.Email.From, err.Error())
		return mailboxLeave
	}
	source := reportSource(m.Email, forward, replied, rids)
	if source != "" && !recordReports(m.Email, rids, source) {
		return mailboxLeave
	}
	if !replied && source == "" {
		log.Debugf("Leaving non-campaign mail unread: %s", m.Email.Subject)
		return mailboxLeave
	}
	if im.DeleteReportedCampaignEmail && source != "" {
		return mailboxDelete
	}
	return mailboxRead
}

func processSenderMailboxMessage(im models.IMAP, m Email, senders map[string]bool) bool {
	if im.RestrictDomain != "" {
		domain := senderDomain(m.Email.From)
		if !strings.EqualFold(domain, im.RestrictDomain) {
			return false
		}
	}
	if isCampaignOriginal(m.Email, senders) || isForwardSubject(m.Email.Subject) {
		return false
	}
	if !handleReplyEmail(m.Email) {
		return false
	}
	rids, err := matchEmail(m.Email)
	if err != nil {
		log.Errorf("Error searching reply for rids from user '%s': %s", m.Email.From, err.Error())
		return true
	}
	if isComplaintReply(m.Email) && len(rids) > 0 && !recordReports(m.Email, rids, reportSourceComplaint) {
		return false
	}
	return true
}

func checkRIDs(em *email.Email, rids map[string]bool) {
	addRIDs(string(em.Text)+string(em.HTML), rids)
}

func addRIDs(content string, rids map[string]bool) {
	scanRIDs(content, rids)
	if decoded, err := url.QueryUnescape(content); err == nil && decoded != content {
		scanRIDs(decoded, rids)
		if decodedAgain, err := url.QueryUnescape(decoded); err == nil {
			scanRIDs(decodedAgain, rids)
		}
	}
}

func scanRIDs(content string, rids map[string]bool) {
	for _, r := range goPhishRegex.FindAllStringSubmatch(content, -1) {
		newrid := r[len(r)-1]
		if !rids[newrid] {
			rids[newrid] = true
		}
	}
}

// handleReplyEmail records an Email Replied event when In-Reply-To / References
// match a previously sent campaign Message-ID and the From address matches.
func handleReplyEmail(em *email.Email) bool {
	ids := extractThreadMessageIDs(em)
	if len(ids) == 0 {
		return false
	}
	fromAddr := extractEmailAddress(em.From)
	for _, id := range ids {
		result, err := models.GetResultByMessageID(id)
		if err != nil {
			continue
		}
		if fromAddr != "" && !strings.EqualFold(fromAddr, result.Email) {
			log.Debugf("Ignoring reply Message-ID match: from %s != result %s", fromAddr, result.Email)
			continue
		}
		log.Infof("User '%s' replied to campaign email rid %s", em.From, result.RId)
		if err := result.HandleEmailReply(models.EventDetails{}); err != nil {
			log.Error("Error recording email reply for rid ", result.RId, ": ", err.Error())
			return false
		}
		return true
	}
	return false
}

func extractThreadMessageIDs(em *email.Email) []string {
	var ids []string
	if em.Headers == nil {
		return ids
	}
	for _, key := range []string{"In-Reply-To", "References"} {
		for _, raw := range em.Headers[textprotoCanonical(key)] {
			ids = append(ids, splitMessageIDs(raw)...)
		}
		// Also try non-canonical lookup via Get
		if v := em.Headers.Get(key); v != "" {
			ids = append(ids, splitMessageIDs(v)...)
		}
	}
	return uniqueStrings(ids)
}

func textprotoCanonical(s string) string {
	// MIMEHeader keys are typically in Canonical MIME form
	parts := strings.Split(s, "-")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	return strings.Join(parts, "-")
}

func splitMessageIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Fields(raw) {
		part = strings.Trim(part, ",")
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		k := models.NormalizeMessageID(s)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, s)
	}
	return out
}

// isForwardSubject reports whether the mailbox client marked the message
// as a forward, after peeling reply prefixes such as "Re:" or "回复:".
func isForwardSubject(subject string) bool {
	folded := strings.ToLower(subjectWithoutReplyPrefixes(subject))
	for _, prefix := range []string{
		"fwd:", "fwd：", "fw:", "fw：",
		"转发:", "转发：", "轉寄:", "轉寄：",
		"[转发]", "【转发】", "[轉寄]", "【轉寄】",
		"自动转发:", "自动转发：", "自动轉寄:", "自动轉寄：",
	} {
		if strings.HasPrefix(folded, prefix) {
			return true
		}
	}
	return false
}

func subjectWithoutReplyPrefixes(subject string) string {
	s := strings.TrimSpace(subject)
	prefixes := []string{"re:", "re：", "回复:", "回复：", "答复:", "答复：", "回覆:", "回覆："}
	for {
		folded := strings.ToLower(s)
		stripped := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(folded, prefix) {
				s = strings.TrimSpace(s[len(prefix):])
				stripped = true
				break
			}
		}
		if !stripped {
			return s
		}
	}
}

func isComplaintReply(em *email.Email) bool {
	text := newReplyText(string(em.Text), string(em.HTML))
	for _, word := range []string{"钓鱼", "可疑", "举报", "诈骗"} {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func newReplyText(text, html string) string {
	plain := text
	if strings.TrimSpace(plain) == "" {
		plain = stripTags(html)
	}
	var b strings.Builder
	for _, line := range strings.Split(plain, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(trimmed, ">") || strings.Contains(trimmed, "写道") || strings.Contains(lower, "wrote:") {
			break
		}
		b.WriteString(trimmed)
		b.WriteByte('\n')
	}
	return b.String()
}

func stripTags(html string) string {
	var b strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func hasWrappedOriginal(em *email.Email) bool {
	for _, a := range em.Attachments {
		if attachmentIsRFC822(a) {
			return true
		}
	}
	return false
}

func attachmentIsRFC822(a *email.Attachment) bool {
	mediaType := a.Header.Get("Content-Type")
	if parsed, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsed
	}
	ext := strings.ToLower(filepath.Ext(a.Filename))
	return mediaType == "message/rfc822" || ext == ".eml"
}

func attachmentIsCalendar(a *email.Attachment) bool {
	mediaType := a.Header.Get("Content-Type")
	if parsed, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsed
	}
	ext := strings.ToLower(filepath.Ext(a.Filename))
	return mediaType == "text/calendar" || ext == ".ics"
}

func reportSource(em *email.Email, forward, replied bool, rids map[string]bool) string {
	if len(rids) == 0 {
		return ""
	}
	if forward {
		return reportSourceForward
	}
	if hasWrappedOriginal(em) {
		return reportSourceAttachment
	}
	if replied && isComplaintReply(em) {
		return reportSourceComplaint
	}
	return ""
}

func recordReports(em *email.Email, rids map[string]bool, source string) bool {
	ok := true
	for rid := range rids {
		result, err := models.GetResult(rid)
		if err != nil {
			log.Error("Error reporting GoPhish email with rid ", rid, ": ", err.Error())
			ok = false
			continue
		}
		details := models.EventDetails{
			From:    em.From,
			Subject: em.Subject,
			Source:  source,
		}
		if em.Headers != nil {
			details.MessageID = em.Headers.Get("Message-ID")
		}
		if err := result.HandleEmailReport(details); err != nil {
			log.Error("Error updating GoPhish email with rid ", rid, ": ", err.Error())
			ok = false
			continue
		}
		log.Infof("User '%s' reported email with rid %s (%s)", em.From, rid, source)
	}
	return ok
}

func isCampaignOriginal(em *email.Email, senders map[string]bool) bool {
	from := strings.ToLower(extractEmailAddress(em.From))
	if from != "" && senders[from] {
		return true
	}
	if em.Headers == nil {
		return false
	}
	id := em.Headers.Get("Message-ID")
	if id == "" {
		return false
	}
	_, err := models.GetResultByMessageID(id)
	return err == nil
}

func sendingAddresses(uid int64) map[string]bool {
	out := map[string]bool{}
	smtps, err := models.GetSMTPs(uid)
	if err != nil {
		log.Error(err)
		return out
	}
	for _, smtp := range smtps {
		if addr := strings.ToLower(extractEmailAddress(smtp.FromAddress)); addr != "" {
			out[addr] = true
		}
		if addr := strings.ToLower(extractEmailAddress(smtp.Username)); strings.Contains(addr, "@") {
			out[addr] = true
		}
	}
	return out
}

func senderDomain(from string) string {
	addr := extractEmailAddress(from)
	at := strings.LastIndex(addr, "@")
	if at < 0 || at == len(addr)-1 {
		return ""
	}
	return strings.ToLower(addr[at+1:])
}

func extractEmailAddress(from string) string {
	from = strings.TrimSpace(from)
	if from == "" {
		return ""
	}
	if i := strings.Index(from, "<"); i >= 0 {
		j := strings.Index(from[i:], ">")
		if j > 1 {
			return strings.TrimSpace(from[i+1 : i+j])
		}
	}
	return from
}

// returns a slice of gophish rid paramters found in the email HTML, Text, and attachments
func matchEmail(em *email.Email) (map[string]bool, error) {
	rids := make(map[string]bool)
	checkRIDs(em, rids)

	// Next check each attachment
	for _, a := range em.Attachments {
		ext := strings.ToLower(filepath.Ext(a.Filename))
		if attachmentIsRFC822(a) {
			rawBodyStream := bytes.NewReader(a.Content)
			attachmentEmail, err := email.NewEmailFromReader(rawBodyStream)
			if err != nil {
				return rids, err
			}
			checkRIDs(attachmentEmail, rids)
			continue
		}
		if attachmentIsCalendar(a) || ext == ".ics" {
			addRIDs(string(a.Content), rids)
		}
	}

	return rids, nil
}
