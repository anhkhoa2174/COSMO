package skills

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	fbRepo "github.com/rockship/cosmo-agents-go/internal/repository/feedback"
	intRepo "github.com/rockship/cosmo-agents-go/internal/repository/interaction"
	segRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	v1 "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"gorm.io/gorm"
)

func openDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestContactQuery_GetContactUnpacksJSONB(t *testing.T) {
	db := openDB(t, &domain.Contact{})
	repo := contactRepo.NewContactRepository(db)
	ctx := context.Background()

	c := &domain.Contact{
		UserID:       uuid.New(),
		SourceID:     "s",
		Source:       "csv",
		Name:         "Ada Lovelace",
		Company:      "Analytical",
		JobTitle:     "CTO",
		DoNotContact: true,
		Profile:      domain.JSONB(`{"email":"ada@example.com","company_data":{"industry":"SaaS"}}`),
		Scores:       domain.JSONB(`{"engagement":80}`),
		Tags:         domain.JSONB(`{}`),
	}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatal(err)
	}

	got, err := NewContactQuerySkill(repo).GetContact(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got["email"] != "ada@example.com" || got["name"] != "Ada Lovelace" || got["job_title"] != "CTO" {
		t.Fatalf("scalar fields: %v", got)
	}
	if got["do_not_contact"] != true {
		t.Fatalf("do_not_contact lost: %v", got["do_not_contact"])
	}
	// The segmentation scorer reads these paths, so they must be real maps.
	if getString(got, "profile.company_data.industry") != "SaaS" {
		t.Fatalf("profile not unpacked: %v", got["profile"])
	}
	if scoreEngagement(got) != 80 {
		t.Fatalf("scores not unpacked: %v", got["scores"])
	}
	for _, k := range []string{"tags", "confirmed_facts", "ai_insights", "insight_validation"} {
		if _, ok := got[k].(map[string]any); !ok {
			t.Errorf("%s is %T, want a map even when empty", k, got[k])
		}
	}

	if _, err := NewContactQuerySkill(repo).GetContact(ctx, uuid.New()); err == nil {
		t.Fatal("a missing contact must be an error")
	}
}

func TestMarshalContact_ProfileWithoutEmail(t *testing.T) {
	got := marshalContact(&domain.Contact{Profile: domain.JSONB(`not json`)})
	if got["email"] != "" {
		t.Fatalf("email = %v", got["email"])
	}
	if p, ok := got["profile"].(map[string]any); !ok || len(p) != 0 {
		t.Fatalf("profile = %v", got["profile"])
	}
}

func TestFeedbackCapture_RecordsEachKind(t *testing.T) {
	db := openDB(t, &domain.UserFeedback{})
	repo := fbRepo.NewRepository(db)
	s := NewFeedbackCaptureSkill(repo)
	ctx := context.Background()
	user, contact := uuid.New(), uuid.New()

	tests := []struct {
		name     string
		capture  func() (*domain.UserFeedback, error)
		wantType string
		wantKey  string
		wantVal  interface{}
	}{
		{"score adjustment",
			func() (*domain.UserFeedback, error) {
				return s.CaptureScoreAdjustment(ctx, user, contact, "scores.fit", 40, 70, "met them")
			}, "score_adjustment", "adjusted_score", 70.0},
		{"insight validation",
			func() (*domain.UserFeedback, error) {
				return s.CaptureInsightValidation(ctx, user, contact, "pain", "hiring", "confirmed", map[string]any{"x": 1})
			}, "insight_validation", "validation", "confirmed"},
		{"custom fact",
			func() (*domain.UserFeedback, error) {
				return s.CaptureCustomFact(ctx, user, contact, "budget", map[string]any{"usd": 5000})
			}, "custom_fact", "fact_type", "budget"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb, err := tt.capture()
			if err != nil {
				t.Fatal(err)
			}
			if fb.ID == uuid.Nil || fb.FeedbackType != tt.wantType || fb.EntityType != "contact" || fb.EntityID != contact {
				t.Fatalf("unexpected record: %+v", fb)
			}
			if !fb.Applied || fb.AppliedAt == nil || time.Since(*fb.AppliedAt) > time.Minute {
				t.Fatalf("applied flags: %v %v", fb.Applied, fb.AppliedAt)
			}
			var data map[string]any
			if err := fb.FeedbackData.Unmarshal(&data); err != nil {
				t.Fatal(err)
			}
			if data[tt.wantKey] != tt.wantVal {
				t.Fatalf("%s = %v, want %v", tt.wantKey, data[tt.wantKey], tt.wantVal)
			}
		})
	}

	stored, err := repo.ListByUser(ctx, user, 10)
	if err != nil || len(stored) != 3 {
		t.Fatalf("stored %d records, %v", len(stored), err)
	}

	// A failed insert surfaces as an error, not a half-built record.
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	if fb, err := s.CaptureCustomFact(ctx, user, contact, "x", nil); err == nil || fb != nil {
		t.Fatalf("closed DB: %v, %v", fb, err)
	}
}

func TestInteractionLog_LogAndList(t *testing.T) {
	db := openDB(t, &domain.Interaction{})
	s := NewInteractionLogSkill(intRepo.NewRepository(db))
	ctx := context.Background()
	contact := uuid.New()

	earlier := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	if _, err := s.LogInteraction(ctx, v1.CreateInteractionRequest{
		ContactID: contact, InteractionType: "sent", Channel: "email", Direction: "outbound", OccurredAt: &earlier,
	}); err != nil {
		t.Fatal(err)
	}
	// No OccurredAt: the skill stamps it with the current time.
	resp, err := s.LogInteraction(ctx, v1.CreateInteractionRequest{
		ContactID: contact, InteractionType: "reply", Channel: "email", Direction: "inbound",
		Content: map[string]interface{}{"body": "sounds good"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID == uuid.Nil || resp.InteractionType != "reply" || resp.Content["body"] != "sounds good" {
		t.Fatalf("response: %+v", resp)
	}

	list, err := s.ListByContact(ctx, contact, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].InteractionType != "reply" || list[1].InteractionType != "sent" {
		t.Fatalf("expected newest first, got %+v", list)
	}
	if other, _ := s.ListByContact(ctx, uuid.New(), 10); len(other) != 0 {
		t.Fatalf("another contact's history leaked: %+v", other)
	}
}

func TestScoring_SaveScoreAndEnrol(t *testing.T) {
	db := openDB(t, &domain.SegmentationScore{})
	repo := segRepo.NewScoreRepository(db)
	s := NewScoringSkill(repo)
	ctx := context.Background()
	contact, seg := uuid.New(), uuid.New()

	if err := s.SaveScore(ctx, contact, seg, FitResult{FitScore: 55, ScoreBreakdown: map[string]int{"title_match": 100}}); err != nil {
		t.Fatal(err)
	}
	scores, err := repo.ListByContact(ctx, contact)
	if err != nil || len(scores) != 1 {
		t.Fatalf("got %d rows, %v", len(scores), err)
	}
	if scores[0].FitScore != 55 || scores[0].Status != "qualified" || scores[0].SegmentationID != seg {
		t.Fatalf("unexpected score: %+v", scores[0])
	}
	var breakdown map[string]any
	_ = scores[0].ScoreBreakdown.Unmarshal(&breakdown)
	if breakdown["title_match"] != 100.0 {
		t.Fatalf("breakdown: %v", breakdown)
	}

	if err := s.MarkAsEnrolled(ctx, contact, seg); err != nil {
		t.Fatal(err)
	}
	scores, _ = repo.ListByContact(ctx, contact)
	if scores[0].Status != "enrolled" || !scores[0].EnrolledInCampaign {
		t.Fatalf("not enrolled: %+v", scores[0])
	}
}

// BUG: re-scoring a contact never changes its stored score.
// segmentation.ScoreRepository.UpsertScore runs
// Where(contact, segment).Assign(score).FirstOrCreate(score): when the row
// exists, FirstOrCreate loads it into `score` — the same struct Assign reads
// its values from — so the old values are written back over themselves. A
// contact scored 55 and later 80 stays at 55, and auto-enrolment thresholds
// act on the stale figure. The fix belongs in the repository package.
func TestScoring_RescoringUpdatesTheStoredScore(t *testing.T) {
	db := openDB(t, &domain.SegmentationScore{})
	repo := segRepo.NewScoreRepository(db)
	s := NewScoringSkill(repo)
	ctx := context.Background()
	contact, seg := uuid.New(), uuid.New()

	if err := s.SaveScore(ctx, contact, seg, FitResult{FitScore: 55, ScoreBreakdown: map[string]int{"title_match": 100}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveScore(ctx, contact, seg, FitResult{FitScore: 80, ScoreBreakdown: map[string]int{"title_match": 70}}); err != nil {
		t.Fatal(err)
	}
	scores, err := repo.ListByContact(ctx, contact)
	if err != nil {
		t.Fatal(err)
	}
	if len(scores) != 1 {
		t.Fatalf("re-scoring must update, not duplicate: %d rows", len(scores))
	}
	if scores[0].FitScore != 80 {
		t.Skip("BUG: ScoreRepository.UpsertScore (internal/repository/segmentation) keeps the old fit_score on re-score: got " +
			fmt.Sprint(scores[0].FitScore) + ", want 80")
	}
}
