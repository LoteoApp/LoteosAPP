package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func square(offset float64) domain.Polygon {
	return domain.Polygon{
		{X: offset, Y: offset},
		{X: offset + 10, Y: offset},
		{X: offset + 10, Y: offset + 10},
		{X: offset, Y: offset + 10},
	}
}

func administrador() loteos.Actor {
	return loteos.Actor{AuthProviderID: "actor-1", Roles: []string{domain.RolAdministrador}}
}

func planWithTwoManzanas() *loteos.PlanInput {
	return &loteos.PlanInput{
		Loteo: loteos.EntityInput{Handle: "1A", Polygon: square(0)},
		Manzanas: []loteos.ManzanaInput{
			{EntityInput: loteos.EntityInput{Handle: "2A", Polygon: square(1)}, Ref: "MANZANA-0"},
			{EntityInput: loteos.EntityInput{Handle: "2B", Polygon: square(2)}, Ref: "MANZANA-1"},
		},
		Lotes: []loteos.LoteInput{
			{EntityInput: loteos.EntityInput{Handle: "3A", Polygon: square(3)}, ManzanaRef: "MANZANA-1"},
			{EntityInput: loteos.EntityInput{Handle: "3B", Polygon: square(4)}, ManzanaRef: "MANZANA-0"},
		},
		Calles: []loteos.EntityInput{
			{Handle: "4A", Polygon: square(5)},
		},
	}
}

func TestCreateLoteoRejectsCallersWithoutTheAdministradorRole(t *testing.T) {
	for _, rol := range []string{domain.RolAgrimensor, domain.RolAdministrativo, domain.RolEscribano, domain.RolInmobiliaria} {
		t.Run(rol, func(t *testing.T) {
			repository := &gatewayfake.LoteoRepository{}
			useCase := loteos.NewCreateLoteo(repository)

			actor := loteos.Actor{AuthProviderID: "actor-1", Roles: []string{rol}}
			_, err := useCase.Execute(context.Background(), actor, loteos.LoteoInput{Name: "Loteo Norte"})

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if repository.CreateCalls != 0 {
				t.Error("Execute() should not reach the repository when the caller has no permission")
			}
		})
	}
}

func TestCreateLoteoRequiresAName(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewCreateLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{Name: "   "})

	if !errors.Is(err, domain.ErrInvalidLoteoName) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidLoteoName)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not reach the repository with an invalid name")
	}
}

func TestCreateLoteoAcceptsALoteoWithoutAPlan(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewCreateLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{
		Name: "  Loteo Norte  ", Location: "  Córdoba  ", Description: "  Al norte  ",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if repository.ReceivedLoteo.Plan != nil {
		t.Error("Execute() should pass a nil plan when the caller sent none")
	}
	if repository.ReceivedLoteo.Name != "Loteo Norte" {
		t.Errorf("Name = %q, want the trimmed value", repository.ReceivedLoteo.Name)
	}
	if repository.ReceivedLoteo.Location != "Córdoba" || repository.ReceivedLoteo.Description != "Al norte" {
		t.Errorf("Execute() should trim every text field, got %#v", repository.ReceivedLoteo)
	}
}

func TestCreateLoteoPassesTheActorToTheRepository(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewCreateLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{Name: "Loteo Norte"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if repository.ActorAuthProviderID != "actor-1" {
		t.Errorf("actor = %q, want the caller's auth provider id", repository.ActorAuthProviderID)
	}
}

func TestCreateLoteoResolvesEachLoteToItsManzana(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewCreateLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{
		Name: "Loteo Norte", Plan: planWithTwoManzanas(),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	plan := repository.ReceivedLoteo.Plan
	if plan == nil {
		t.Fatal("Execute() should pass the plan to the repository")
	}
	if len(plan.Manzanas) != 2 || len(plan.Lotes) != 2 || len(plan.Calles) != 1 {
		t.Fatalf("plan = %d manzanas, %d lotes, %d calles", len(plan.Manzanas), len(plan.Lotes), len(plan.Calles))
	}
	// The first lote named MANZANA-1, the second MANZANA-0, so the resolved
	// positions must not simply follow the order the lotes arrived in.
	if plan.Lotes[0].ManzanaIndex != 1 {
		t.Errorf("lote[0].ManzanaIndex = %d, want 1", plan.Lotes[0].ManzanaIndex)
	}
	if plan.Lotes[1].ManzanaIndex != 0 {
		t.Errorf("lote[1].ManzanaIndex = %d, want 0", plan.Lotes[1].ManzanaIndex)
	}
	if plan.Loteo.Handle != "1A" {
		t.Errorf("loteo handle = %q, want 1A", plan.Loteo.Handle)
	}
}

func TestCreateLoteoRejectsALoteThatNamesAManzanaOutsideThePlan(t *testing.T) {
	plan := planWithTwoManzanas()
	plan.Lotes[0].ManzanaRef = "MANZANA-DE-OTRO-LOTEO"

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewCreateLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{
		Name: "Loteo Norte", Plan: plan,
	})

	if !errors.Is(err, domain.ErrUnknownManzana) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUnknownManzana)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not persist a plan whose hierarchy doesn't resolve")
	}
}

func TestCreateLoteoRejectsAmbiguousManzanaReferences(t *testing.T) {
	tests := map[string]func(*loteos.PlanInput){
		"duplicate": func(plan *loteos.PlanInput) { plan.Manzanas[1].Ref = plan.Manzanas[0].Ref },
		"empty":     func(plan *loteos.PlanInput) { plan.Manzanas[1].Ref = "  " },
	}

	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			plan := planWithTwoManzanas()
			corrupt(plan)

			repository := &gatewayfake.LoteoRepository{}
			useCase := loteos.NewCreateLoteo(repository)

			_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{
				Name: "Loteo Norte", Plan: plan,
			})

			if !errors.Is(err, domain.ErrInvalidManzanaRef) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidManzanaRef)
			}
			if repository.CreateCalls != 0 {
				t.Error("Execute() should not persist a plan with ambiguous references")
			}
		})
	}
}

func TestCreateLoteoRejectsAPlanWithoutTheLoteoPolygon(t *testing.T) {
	plan := planWithTwoManzanas()
	plan.Loteo.Polygon = nil

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewCreateLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{
		Name: "Loteo Norte", Plan: plan,
	})

	if !errors.Is(err, domain.ErrPlanWithoutLoteo) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPlanWithoutLoteo)
	}
}

func TestCreateLoteoRejectsAnUnusablePolygonOnAnyLayer(t *testing.T) {
	openRing := domain.Polygon{{X: 0, Y: 0}, {X: 1, Y: 1}}

	tests := map[string]func(*loteos.PlanInput){
		"loteo":   func(plan *loteos.PlanInput) { plan.Loteo.Polygon = openRing },
		"manzana": func(plan *loteos.PlanInput) { plan.Manzanas[0].Polygon = openRing },
		"lote":    func(plan *loteos.PlanInput) { plan.Lotes[0].Polygon = openRing },
		"calle":   func(plan *loteos.PlanInput) { plan.Calles[0].Polygon = openRing },
	}

	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			plan := planWithTwoManzanas()
			corrupt(plan)

			repository := &gatewayfake.LoteoRepository{}
			useCase := loteos.NewCreateLoteo(repository)

			_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{
				Name: "Loteo Norte", Plan: plan,
			})

			if !errors.Is(err, domain.ErrInvalidGeometry) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidGeometry)
			}
			if repository.CreateCalls != 0 {
				t.Error("Execute() should not persist a plan with an unusable polygon")
			}
		})
	}
}

func TestCreateLoteoRejectsAPlanWithTooManyPolygons(t *testing.T) {
	plan := planWithTwoManzanas()
	plan.Lotes = make([]loteos.LoteInput, domain.MaxPolygonsPerPlan)

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewCreateLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{
		Name: "Loteo Norte", Plan: plan,
	})

	if !errors.Is(err, domain.ErrPlanTooLarge) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPlanTooLarge)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should reject an oversized plan before validating it polygon by polygon")
	}
}

func TestCreateLoteoRejectsAPlanWithTooManyVertices(t *testing.T) {
	// Few enough polygons to pass the count limit, but together far more
	// vertices than the plan limit: checking a ring is quadratic in its
	// vertices, so this has to be rejected before any ring is validated.
	ring := make(domain.Polygon, domain.MaxVerticesPerPolygon)
	plan := planWithTwoManzanas()
	plan.Lotes = make([]loteos.LoteInput, domain.MaxVerticesPerPlan/domain.MaxVerticesPerPolygon+1)
	for i := range plan.Lotes {
		plan.Lotes[i] = loteos.LoteInput{
			EntityInput: loteos.EntityInput{Polygon: ring},
			ManzanaRef:  "MANZANA-0",
		}
	}

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewCreateLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{
		Name: "Loteo Norte", Plan: plan,
	})

	if !errors.Is(err, domain.ErrPlanTooLarge) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPlanTooLarge)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should reject an oversized plan before validating it polygon by polygon")
	}
}

func TestCreateLoteoReportsAnUnexpectedRepositoryFailureAsUnavailable(t *testing.T) {
	cause := errors.New("connection refused")
	repository := &gatewayfake.LoteoRepository{CreateErr: cause}
	useCase := loteos.NewCreateLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{Name: "Loteo Norte"})

	assertUnavailable(t, err, cause)
}

func TestCreateLoteoKeepsABusinessErrorFromTheRepository(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{CreateErr: domain.ErrInvalidLoteoName}
	useCase := loteos.NewCreateLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), loteos.LoteoInput{Name: "Loteo Norte"})

	if !errors.Is(err, domain.ErrInvalidLoteoName) {
		t.Fatalf("Execute() error = %v, want the repository's business error", err)
	}
}

// assertUnavailable checks that an infrastructure failure reaches the caller
// classified as unavailable, with the original error kept as Cause so it can
// be logged without being shown.
func assertUnavailable(t *testing.T, err error, cause error) {
	t.Helper()

	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("Execute() error = %v, want a *domain.Error", err)
	}
	if domainErr.Kind != domain.KindUnavailable {
		t.Errorf("error kind = %q, want %q", domainErr.Kind, domain.KindUnavailable)
	}
	if !errors.Is(err, cause) {
		t.Errorf("error = %v, want it to carry the repository failure as cause", err)
	}
	if domainErr.Message == cause.Error() {
		t.Error("the message shown to the caller should not be the internal failure")
	}
}
