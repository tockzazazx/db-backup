// Package notify sends backup result emails through the team's email API.
package notify

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const (
	apiURL      = "https://box.one.th/admin/api/v2/send_email_by_template_with_cc"
	senderEmail = "onebox@inet.co.th"
)

// Send posts one plain-text email to every address in to.
func Send(ctx context.Context, token string, to []string, subject, content string) error {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	for field, value := range map[string]string{
		"sender_email":   senderEmail,
		"receiver_email": receiverList(to),
		"subject_email":  subject,
		"content_email":  content,
		"cc_email":       "{}",
	} {
		if err := w.WriteField(field, value); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("email API unreachable: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("email API returned %s: %s", resp.Status, bytes.TrimSpace(respBody))
	}
	return nil
}

// receiverList renders the API's set-literal format: {'a@x.com','b@y.com'}.
func receiverList(to []string) string {
	quoted := make([]string, 0, len(to))
	for _, t := range to {
		if t = strings.TrimSpace(t); t != "" {
			quoted = append(quoted, "'"+t+"'")
		}
	}
	return "{" + strings.Join(quoted, ",") + "}"
}

// File is one uploaded file in a Report.
type File struct {
	Name string
	Size int64
}

// Report summarizes one upload run for the notification email.
type Report struct {
	Host     string
	Dest     string // S3 destination, e.g. "ubuntu-tock/2026-07-16"
	Uploaded []File
	Skipped  int
	Started  time.Time
	Duration time.Duration
	Err      error
}

// Subject is the email subject line for this report.
func (r Report) Subject() string {
	status := "สำเร็จ"
	if r.Err != nil {
		status = "ล้มเหลว"
	}
	return fmt.Sprintf("[boxdb] backup %s - %s - %s", status, r.Host, r.Started.Format("2006-01-02"))
}

// Content is the plain-text email body for this report.
func (r Report) Content() string {
	var b strings.Builder
	b.WriteString("สรุปผลการ backup (boxdb upload)\n\n")
	fmt.Fprintf(&b, "เครื่อง       : %s\n", r.Host)
	if r.Dest != "" {
		fmt.Fprintf(&b, "ปลายทาง S3   : %s\n", r.Dest)
	}
	fmt.Fprintf(&b, "เวลาเริ่ม     : %s\n", r.Started.Format("2006-01-02 15:04:05 (MST)"))
	fmt.Fprintf(&b, "ใช้เวลา       : %s\n", r.Duration.Round(time.Second))
	if r.Err != nil {
		fmt.Fprintf(&b, "ผลลัพธ์       : ล้มเหลว\nสาเหตุ        : %v\n", r.Err)
	} else {
		b.WriteString("ผลลัพธ์       : สำเร็จ\n")
	}

	b.WriteString("\n")
	if len(r.Uploaded) == 0 {
		b.WriteString("ไม่มีไฟล์ใหม่ให้อัพโหลดในรอบนี้\n")
	} else {
		var total int64
		for _, f := range r.Uploaded {
			total += f.Size
		}
		fmt.Fprintf(&b, "ไฟล์ที่อัพโหลดใหม่ %d ไฟล์ (รวม %s):\n", len(r.Uploaded), humanSize(total))
		for _, f := range r.Uploaded {
			fmt.Fprintf(&b, "  - %s (%s)\n", f.Name, humanSize(f.Size))
		}
	}
	fmt.Fprintf(&b, "ข้ามไฟล์ที่เคยอัพโหลดแล้ว %d ไฟล์\n", r.Skipped)
	return b.String()
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTP"[exp])
}
