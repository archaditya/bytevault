package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/notification/email"
	"github.com/archaditya/bytevault/internal/repository"
)

type ContactService struct {
	contactRepo *repository.ContactRepository
	emailClient *email.BrevoClient
}

func NewContactService(contactRepo *repository.ContactRepository, emailClient *email.BrevoClient) *ContactService {
	return &ContactService{
		contactRepo: contactRepo,
		emailClient: emailClient,
	}
}

func (s *ContactService) SubmitQuery(ctx context.Context, q *model.ContactQuery) error {
	if q.Name == "" || q.Email == "" || q.Subject == "" || q.Message == "" {
		return errors.New("name, email, subject, and message are required fields")
	}
	return s.contactRepo.Create(ctx, q)
}

func (s *ContactService) ListQueries(ctx context.Context, limit, offset int) ([]*model.ContactQuery, int, error) {
	return s.contactRepo.ListAll(ctx, limit, offset)
}

func (s *ContactService) ReplyToQuery(ctx context.Context, id string, reply string, repliedBy string) error {
	if reply == "" {
		return errors.New("reply message cannot be empty")
	}
	q, err := s.contactRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if q == nil {
		return errors.New("contact query not found")
	}

	// 1. Save response in the database
	err = s.contactRepo.UpdateReply(ctx, id, reply, repliedBy)
	if err != nil {
		return err
	}

	// 2. Dispatch email to user if email integration is configured
	if s.emailClient != nil {
		subject := fmt.Sprintf("Re: %s", q.Subject)
		emailBody := fmt.Sprintf(`
			<div style="font-family: sans-serif; line-height: 1.6; color: #1f2937; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #e5e7eb; border-radius: 8px;">
				<h2 style="color: #5e6ad2; margin-top: 0;">Support Ticket Reply</h2>
				<p>Hello <strong>%s</strong>,</p>
				<p>We have replied to your contact query regarding <strong>"%s"</strong>:</p>
				<div style="background-color: #f9fafb; padding: 16px; border-left: 4px solid #5e6ad2; margin: 16px 0; border-radius: 4px; font-style: italic;">
					%s
				</div>
				<p>If you have any further questions, you can respond to this email or submit another query.</p>
				<hr style="border: 0; border-top: 1px solid #e5e7eb; margin: 24px 0;" />
				<p style="font-size: 11px; color: #9ca3af;">
					Your original inquiry:<br/>
					"%s"
				</p>
			</div>
		`, q.Name, q.Subject, reply, q.Message)

		// Send asynchronously in a background goroutine so the API response remains fast
		go func() {
			_ = s.emailClient.SendGeneric(context.Background(), q.Email, q.Name, subject, emailBody)
		}()
	}

	return nil
}
