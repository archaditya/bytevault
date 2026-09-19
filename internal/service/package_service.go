package service

import (
	"context"
	"fmt"
	"math"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/razorpay"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/rs/zerolog/log"
)

type PackageService struct {
	pkgRepo        *repository.PackageRepository
	razorpayClient *razorpay.Client
}

func NewPackageService(pkgRepo *repository.PackageRepository, razorpayClient *razorpay.Client) *PackageService {
	return &PackageService{
		pkgRepo:        pkgRepo,
		razorpayClient: razorpayClient,
	}
}

func (s *PackageService) ListPackages(ctx context.Context, onlyActive bool) ([]*model.Package, error) {
	return s.pkgRepo.FindAll(ctx, onlyActive)
}

func (s *PackageService) GetPackageByID(ctx context.Context, id string) (*model.Package, error) {
	return s.pkgRepo.FindByID(ctx, id)
}

func (s *PackageService) GetPackageByName(ctx context.Context, name string) (*model.Package, error) {
	return s.pkgRepo.FindByName(ctx, name)
}

// CreatePackage creates a package tier. If it's a paid tier and Razorpay client is active,
// it automatically creates the corresponding Razorpay billing plan and links its plan_id.
func (s *PackageService) CreatePackage(ctx context.Context, pkg *model.Package) (*model.Package, error) {
	if pkg.GSTRate <= 0 {
		pkg.GSTRate = 18.00 // Default 18% GST
	}

	// Calculate price with GST in paise
	if pkg.PricePaise > 0 {
		taxAmount := float64(pkg.PricePaise) * (pkg.GSTRate / 100.0)
		pkg.PriceWithGSTPaise = pkg.PricePaise + int(math.Round(taxAmount))

		// If Razorpay client is configured, create Razorpay Plan
		if s.razorpayClient != nil {
			planReq := razorpay.CreatePlanRequest{
				Period:   pkg.BillingPeriod,
				Interval: 1,
				Item: razorpay.PlanItem{
					Name:        pkg.DisplayName,
					Amount:      pkg.PriceWithGSTPaise,
					Currency:    pkg.Currency,
					Description: fmt.Sprintf("ByteVault %s Tier - %d GB Storage", pkg.DisplayName, pkg.StorageLimitBytes/(1024*1024*1024)),
				},
				Notes: map[string]string{
					"package_name": pkg.Name,
				},
			}

			planResp, err := s.razorpayClient.CreatePlan(ctx, planReq)
			if err != nil {
				return nil, fmt.Errorf("create razorpay plan: %w", err)
			}
			pkg.RazorpayPlanID = &planResp.ID
			log.Info().Str("plan_id", planResp.ID).Str("package", pkg.Name).Msg("Created Razorpay plan for package")
		}
	} else {
		pkg.PriceWithGSTPaise = 0
		pkg.RazorpayPlanID = nil
	}

	return s.pkgRepo.Create(ctx, pkg)
}

func (s *PackageService) UpdatePackage(ctx context.Context, pkg *model.Package) error {
	if pkg.PricePaise > 0 {
		taxAmount := float64(pkg.PricePaise) * (pkg.GSTRate / 100.0)
		pkg.PriceWithGSTPaise = pkg.PricePaise + int(math.Round(taxAmount))
	} else {
		pkg.PriceWithGSTPaise = 0
	}
	return s.pkgRepo.Update(ctx, pkg)
}

func (s *PackageService) DeletePackage(ctx context.Context, id string) error {
	return s.pkgRepo.Delete(ctx, id)
}
