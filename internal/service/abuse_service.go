package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/notification/email"
	"github.com/archaditya/bytevault/internal/repository"
)

var uuidRegex = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

type AbuseService struct {
	abuseRepo    *repository.AbuseRepository
	fileRepo     *repository.FileRepository
	userRepo     *repository.UserRepository
	emailClient  *email.BrevoClient
	notifService *NotificationService
}

func NewAbuseService(
	abuseRepo *repository.AbuseRepository,
	fileRepo *repository.FileRepository,
	userRepo *repository.UserRepository,
	emailClient *email.BrevoClient,
) *AbuseService {
	return &AbuseService{
		abuseRepo:   abuseRepo,
		fileRepo:    fileRepo,
		userRepo:    userRepo,
		emailClient: emailClient,
	}
}

func (s *AbuseService) SetNotificationService(ns *NotificationService) {
	s.notifService = ns
}

func (s *AbuseService) SubmitReport(ctx context.Context, req *model.SubmitAbuseReportRequest) (*model.AbuseReport, error) {
	if strings.TrimSpace(req.ReporterName) == "" {
		return nil, errors.New("reporter_name is required")
	}
	if strings.TrimSpace(req.ReporterEmail) == "" {
		return nil, errors.New("reporter_email is required")
	}
	if strings.TrimSpace(req.Category) == "" {
		return nil, errors.New("category is required")
	}
	rawTarget := strings.TrimSpace(req.TargetURL)
	if rawTarget == "" {
		return nil, errors.New("target_url is required")
	}
	if strings.TrimSpace(req.Description) == "" {
		return nil, errors.New("description is required")
	}

	// 1. Explicitly reject ephemeral instant shares and live shares
	lowerTarget := strings.ToLower(rawTarget)
	if strings.Contains(lowerTarget, "/e/instant/") || strings.Contains(lowerTarget, "/s/instant/") ||
		strings.Contains(lowerTarget, "/instant/") || strings.Contains(lowerTarget, "/live/") {
		return nil, errors.New("Instant Shares expire automatically and cannot be reported. Only persistent public shared links (/s/...) can be reported for takedown.")
	}

	if strings.HasPrefix(lowerTarget, "http://") || strings.HasPrefix(lowerTarget, "https://") {
		parsedURL, err := url.Parse(rawTarget)
		if err != nil {
			return nil, errors.New("Invalid URL format")
		}
		host := strings.ToLower(parsedURL.Hostname())
		if host != "pushpostvault.com" && host != "www.pushpostvault.com" && host != "localhost" && host != "127.0.0.1" {
			return nil, errors.New("Only PushPostVault public shared links (https://pushpostvault.com/s/...) can be reported")
		}
		if !strings.HasPrefix(parsedURL.Path, "/s/") {
			return nil, errors.New("Reported link must point to a public shared file (/s/...)")
		}
	}

	// 2. Validate and resolve target file from DB
	matches := uuidRegex.FindStringSubmatch(rawTarget)
	if len(matches) == 0 {
		return nil, errors.New("Invalid shared link format. Please provide a valid PushPostVault shared link (e.g. https://pushpostvault.com/s/xyz)")
	}

	if s.fileRepo == nil {
		return nil, errors.New("File service unavailable")
	}

	candidateID := matches[0]
	file, err := s.fileRepo.FindByID(ctx, candidateID)
	if err != nil || file == nil {
		return nil, errors.New("Reported public shared file does not exist or has already been removed")
	}

	ticket := fmt.Sprintf("PPV-ABU-%X", time.Now().UnixNano()%0xFFFFFFF)

	report := &model.AbuseReport{
		TicketNumber:  ticket,
		FileID:        &file.ID,
		TargetURL:     rawTarget,
		Category:      strings.ToLower(strings.TrimSpace(req.Category)),
		ReporterName:  strings.TrimSpace(req.ReporterName),
		ReporterEmail: strings.TrimSpace(req.ReporterEmail),
		Description:   strings.TrimSpace(req.Description),
		EvidenceURL:   req.EvidenceURL,
		Status:        "pending",
	}

	if err := s.abuseRepo.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to create abuse report: %w", err)
	}

	// Dispatch in-app notification to all administrators
	if s.notifService != nil {
		go func(tNum, cat, reporter, fName string) {
			_ = s.notifService.NotifyAdmins(
				context.Background(),
				"admin.abuse",
				fmt.Sprintf("New Abuse Report: #%s", tNum),
				fmt.Sprintf("%s reported '%s' under %s", reporter, fName, strings.ToUpper(cat)),
				"/admin/moderation?tab=reports",
			)
		}(ticket, report.Category, report.ReporterName, file.Filename)
	}

	// Dispatch statutory acknowledgement email
	if s.emailClient != nil {
		go func(name, toEmail, category, tNum, target string) {
			ackSubject := fmt.Sprintf("[%s] Abuse / DMCA Takedown Notice Lodged", tNum)
			body := fmt.Sprintf(`
				<div style="font-family: sans-serif; line-height: 1.6; color: #1f2937; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e5e7eb; border-radius: 8px;">
					<h2 style="color: #dc2626; margin-top: 0;">Safety & Compliance Center: Ticket Confirmed</h2>
					<p>Hello <strong>%s</strong>,</p>
					<p>We have received your content report regarding an alleged violation under category: <strong>%s</strong>.</p>
					<div style="background-color: #fef2f2; padding: 14px 16px; border-left: 4px solid #dc2626; margin: 16px 0; border-radius: 4px;">
						<strong>Reference Ticket:</strong> %s<br/>
						<strong>Target Resource:</strong> %s<br/>
						<strong>Statutory SLA:</strong> Acknowledged immediately &bull; Investigation &amp; expedited takedown within 24–36 hours upon verification.
					</div>
					<p>Our Trust &amp; Safety Compliance Desk is reviewing the material against our Terms of Service and applicable intermediary guidelines.</p>
					<p style="font-size: 11px; color: #9ca3af; margin-top: 24px;">PushPostVault Safety &amp; Compliance &bull; Rule 3(2) IT Intermediary Rules, 2021</p>
				</div>
			`, name, strings.ToUpper(category), tNum, target)
			_ = s.emailClient.SendGeneric(context.Background(), toEmail, name, ackSubject, body)
		}(report.ReporterName, report.ReporterEmail, report.Category, ticket, report.TargetURL)
	}

	return report, nil
}

func (s *AbuseService) ListReports(ctx context.Context, status string, limit, offset int) ([]*model.AbuseReport, int, error) {
	return s.abuseRepo.ListAll(ctx, status, limit, offset)
}

func (s *AbuseService) GetReport(ctx context.Context, id string) (*model.AbuseReport, error) {
	return s.abuseRepo.FindByID(ctx, id)
}

func (s *AbuseService) TakedownReport(ctx context.Context, id string, adminNotes *string, reviewedBy string) error {
	rep, err := s.abuseRepo.FindByID(ctx, id)
	if err != nil || rep == nil {
		return errors.New("abuse report not found")
	}

	if rep.FileID != nil && s.fileRepo != nil {
		_ = s.fileRepo.UpdateStatus(ctx, *rep.FileID, "BLOCKED_ABUSE")
		_ = s.fileRepo.UpdatePublicStatus(ctx, *rep.FileID, false)
	}

	if err := s.abuseRepo.UpdateStatusAndReview(ctx, id, "actioned_takedown", adminNotes, reviewedBy); err != nil {
		return err
	}

	// Dispatch resolution notification to reporter
	if s.emailClient != nil {
		go func(toEmail, name, tNum string) {
			subject := fmt.Sprintf("[%s] Resolution Notice: Material Disabled", tNum)
			body := fmt.Sprintf(`
				<div style="font-family: sans-serif; line-height: 1.6; color: #1f2937; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e5e7eb; border-radius: 8px;">
					<h2 style="color: #059669; margin-top: 0;">Report Resolution Notice</h2>
					<p>Hello <strong>%s</strong>,</p>
					<p>Regarding your report with Ticket Reference <strong>%s</strong>:</p>
					<div style="background-color: #ecfdf5; padding: 14px 16px; border-left: 4px solid #059669; margin: 16px 0; border-radius: 4px;">
						<strong>Status:</strong> Actioned &amp; Disabled<br/>
						The reported material has been reviewed and taken down from our servers. Public access has been terminated.
					</div>
					<p style="font-size: 11px; color: #9ca3af; margin-top: 24px;">PushPostVault Safety &amp; Compliance</p>
				</div>
			`, name, tNum)
			_ = s.emailClient.SendGeneric(context.Background(), toEmail, name, subject, body)
		}(rep.ReporterEmail, rep.ReporterName, rep.TicketNumber)
	}

	return nil
}

func (s *AbuseService) StrikeUploader(ctx context.Context, id string, adminNotes *string, reviewedBy string) (int, error) {
	rep, err := s.abuseRepo.FindByID(ctx, id)
	if err != nil || rep == nil {
		return 0, errors.New("abuse report not found")
	}

	strikes := 0
	if rep.FileID != nil && s.fileRepo != nil {
		_ = s.fileRepo.UpdateStatus(ctx, *rep.FileID, "BLOCKED_ABUSE")
		_ = s.fileRepo.UpdatePublicStatus(ctx, *rep.FileID, false)

		if s.userRepo != nil && rep.UploaderID != nil {
			var strikeErr error
			strikes, strikeErr = s.userRepo.IncrementNSFWStrikes(ctx, *rep.UploaderID)
			if strikeErr == nil && strikes >= 3 {
				until := time.Now().Add(30 * 24 * time.Hour)
				_ = s.userRepo.RestrictUser(ctx, *rep.UploaderID, &until, "Repeated abuse violations")
			}
		}
	}

	if err := s.abuseRepo.UpdateStatusAndReview(ctx, id, "actioned_takedown", adminNotes, reviewedBy); err != nil {
		return strikes, err
	}

	// Dispatch resolution notification to reporter
	if s.emailClient != nil {
		go func(toEmail, name, tNum string) {
			subject := fmt.Sprintf("[%s] Resolution Notice: Actioned & Sanctioned", tNum)
			body := fmt.Sprintf(`
				<div style="font-family: sans-serif; line-height: 1.6; color: #1f2937; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e5e7eb; border-radius: 8px;">
					<h2 style="color: #059669; margin-top: 0;">Report Resolution Notice</h2>
					<p>Hello <strong>%s</strong>,</p>
					<p>Regarding your report with Ticket Reference <strong>%s</strong>:</p>
					<div style="background-color: #ecfdf5; padding: 14px 16px; border-left: 4px solid #059669; margin: 16px 0; border-radius: 4px;">
						<strong>Status:</strong> Actioned &bull; File Disabled &bull; Uploader Sanctioned<br/>
						The infringing material has been taken down and administrative penalties have been applied to the responsible account.
					</div>
					<p style="font-size: 11px; color: #9ca3af; margin-top: 24px;">PushPostVault Safety &amp; Compliance</p>
				</div>
			`, name, tNum)
			_ = s.emailClient.SendGeneric(context.Background(), toEmail, name, subject, body)
		}(rep.ReporterEmail, rep.ReporterName, rep.TicketNumber)
	}

	return strikes, nil
}

func (s *AbuseService) DismissReport(ctx context.Context, id string, adminNotes *string, reviewedBy string) error {
	rep, err := s.abuseRepo.FindByID(ctx, id)
	if err != nil || rep == nil {
		return errors.New("abuse report not found")
	}

	if err := s.abuseRepo.UpdateStatusAndReview(ctx, id, "dismissed", adminNotes, reviewedBy); err != nil {
		return err
	}

	if s.emailClient != nil {
		go func(toEmail, name, tNum string, notes *string) {
			subject := fmt.Sprintf("[%s] Review Complete: Report Dismissed", tNum)
			noteText := "Upon investigation, the submitted resource did not contain actionable violations or lacked sufficient proof of authorization."
			if notes != nil && *notes != "" {
				noteText = *notes
			}
			body := fmt.Sprintf(`
				<div style="font-family: sans-serif; line-height: 1.6; color: #1f2937; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e5e7eb; border-radius: 8px;">
					<h2 style="color: #4b5563; margin-top: 0;">Report Review Outcome</h2>
					<p>Hello <strong>%s</strong>,</p>
					<p>Regarding your report with Ticket Reference <strong>%s</strong>:</p>
					<div style="background-color: #f3f4f6; padding: 14px 16px; border-left: 4px solid #9ca3af; margin: 16px 0; border-radius: 4px;">
						<strong>Status:</strong> Dismissed<br/>
						%s
					</div>
					<p style="font-size: 11px; color: #9ca3af; margin-top: 24px;">PushPostVault Safety &amp; Compliance</p>
				</div>
			`, name, tNum, noteText)
			_ = s.emailClient.SendGeneric(context.Background(), toEmail, name, subject, body)
		}(rep.ReporterEmail, rep.ReporterName, rep.TicketNumber, adminNotes)
	}

	return nil
}
