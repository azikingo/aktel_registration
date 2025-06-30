package server

import (
	"aktel/internal/database"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/rs/zerolog/log"
)

func (s *FiberServer) RegisterFiberRoutes() {
	// Apply CORS middleware
	s.App.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Accept,Authorization,Content-Type",
		AllowCredentials: false, // credentials require explicit origins
		MaxAge:           300,
	}))

	s.App.Get("/", s.HelloWorldHandler)
	s.App.Post("/aktel-registration", s.registrationHandler)

}

func (s *FiberServer) HelloWorldHandler(c *fiber.Ctx) error {
	resp := fiber.Map{
		"message": "Hello World",
	}

	return c.JSON(resp)
}

type SubmissionResponse struct {
	SubmittedAt       string  `json:"Submission Date"`
	TeamName          string  `json:"Team name"`
	CaptainFirstName  string  `json:"Captain - First Name"`
	CaptainLastName   string  `json:"Captain - Last Name"`
	CaptainGradYear   int32   `json:"Captain - Grad. Year"`
	PhoneNumber       string  `json:"Phone Number"`
	FastCupLink       string  `json:"FastCup Link"`
	Member2FirstName  string  `json:"Member 2 - First Name"`
	Member2LastName   string  `json:"Member 2 - Last Name"`
	Member2GradYear   int32   `json:"Member 2 - Grad. Year"`
	Member3FirstName  string  `json:"Member 3 - First Name"`
	Member3LastName   string  `json:"Member 3 - Last Name"`
	Member3GradYear   int32   `json:"Member 3 - Grad. Year"`
	Member4FirstName  string  `json:"Member 4 - First Name"`
	Member4LastName   string  `json:"Member 4 - Last Name"`
	Member4GradYear   int32   `json:"Member 4 - Grad. Year"`
	Member5FirstName  string  `json:"Member 5 - First Name"`
	Member5LastName   string  `json:"Member 5 - Last Name"`
	Member5GradYear   int32   `json:"Member 5 - Grad. Year"`
	ReserveFirstName  *string `json:"Reserve - First Name"`
	ReserveLastName   *string `json:"Reserve - Last Name"`
	ReserveGradYear   *int32  `json:"Reserve - Grad. Year"`
	TeamLogo          string  `json:"Team logo"`
	SubmissionIP      string  `json:"Submission IP"`
	SubmissionURL     string  `json:"Submission URL"`
	SubmissionEditURL string  `json:"Submission Edit URL"`
	LastUpdateDate    *string `json:"Last Update Date"` // Empty string, keep as string
	SubmissionID      string  `json:"Submission ID"`
}

func (s *FiberServer) registrationHandler(c *fiber.Ctx) error {
	var submission SubmissionResponse
	err := json.Unmarshal(c.Body(), &submission)
	if err != nil {
		s.log.Err(err).Msg("unmarshal body failed")
		return err
	}

	//go s.registerTeam(submission)

	wpMode := "WHATSAPP"
	err = s.wpBot.SendMessage("787471301200", printTeam(submission, &wpMode))
	if err != nil {
		s.log.Err(err).Msg("sending message via whatsapp failed")
	}

	tgMode := tgbotapi.ModeHTML
	_, err = s.tgBot.SendMessageToChannel("@aktel_cs2", printTeam(submission, &tgMode), &tgMode)
	if err != nil {
		s.log.Err(err).Msg("sending message via telegram failed")
	}

	resp := fiber.Map{
		"success": "true",
	}

	return c.JSON(resp)
}

func (s *FiberServer) registerTeam(submission SubmissionResponse) {
	teamDto := database.Team{
		Name:        submission.TeamName,
		FastCupLink: submission.FastCupLink,
		LogoLink:    submission.TeamLogo,
	}

	submittedAt, err := time.Parse(time.RFC3339, submission.SubmittedAt)
	if err != nil {
		s.log.Err(err).Msg("parsing submitted at failed")
	}

	lastUpdatedAt := time.Time{}
	if submission.LastUpdateDate != nil && *submission.LastUpdateDate != "" {
		lastUpdatedAt, err = time.Parse(time.RFC3339, *submission.LastUpdateDate)
		if err != nil {
			s.log.Err(err).Msg("parsing last updated at failed")
		}
	}

	submissionDto := database.Submission{
		ExternalId:        submission.SubmissionID,
		TournamentId:      1,
		SubmittedAt:       submittedAt,
		SubmissionIp:      submission.SubmissionIP,
		SubmissionUrl:     submission.SubmissionURL,
		SubmissionEditUrl: submission.SubmissionEditURL,
		LastUpdatedAt:     lastUpdatedAt,
	}

	phoneNumber, err := normalizePhone(submission.PhoneNumber)
	if err != nil {
		s.log.Err(err).Msg("normalizing phone number failed")
		phoneNumber = submission.PhoneNumber
	}

	membersDto := []database.Member{
		{
			Role:        database.Captain,
			Name:        submission.CaptainFirstName,
			Surname:     submission.CaptainLastName,
			GradYear:    submission.CaptainGradYear,
			PhoneNumber: phoneNumber,
		},
		{
			Role:     database.Player,
			Name:     submission.Member2FirstName,
			Surname:  submission.Member2LastName,
			GradYear: submission.Member2GradYear,
		},
		{
			Role:     database.Player,
			Name:     submission.Member3FirstName,
			Surname:  submission.Member3LastName,
			GradYear: submission.Member3GradYear,
		},
		{
			Role:     database.Player,
			Name:     submission.Member4FirstName,
			Surname:  submission.Member4LastName,
			GradYear: submission.Member4GradYear,
		},
		{
			Role:     database.Player,
			Name:     submission.Member5FirstName,
			Surname:  submission.Member5LastName,
			GradYear: submission.Member5GradYear,
		},
	}

	if submission.ReserveFirstName != nil && submission.ReserveLastName != nil && submission.ReserveGradYear != nil {
		membersDto = append(membersDto, database.Member{
			Role:     database.Reserve,
			Name:     *submission.ReserveFirstName,
			Surname:  *submission.ReserveLastName,
			GradYear: *submission.ReserveGradYear,
		})
	}

	err = s.db.ApplySubmission(context.Background(), teamDto, submissionDto, membersDto)
	if err != nil {
		s.log.Err(err).Msg("team creation failed")
	}

	return
}

// normalizePhone normalizes Kazakhstan phone numbers to the +7XXXXXXXXXX format
func normalizePhone(input string) (reply string, err error) {
	defer func() {
		if err != nil {
			log.Err(err).Msg("number normalize failed")
			reply = input
			err = nil
		}
	}()

	// Remove all non-digit characters
	re := regexp.MustCompile(`\D`)
	digits := re.ReplaceAllString(input, "")

	// Normalize based on common Kazakh formats
	if len(digits) == 11 && strings.HasPrefix(digits, "87") {
		digits = "7" + digits[1:] // convert 8XXXXXXXXXX to 7XXXXXXXXXX
	} else if len(digits) == 10 && strings.HasPrefix(digits, "7") {
		digits = "7" + digits // already correct
	} else if len(digits) == 11 && strings.HasPrefix(digits, "7") {
		// already starting with 7XXXXXXXXXX
	} else {
		return "", fmt.Errorf("invalid phone number format: %s", input)
	}

	// Final validation: should be 11 digits starting with 7
	if len(digits) != 11 || !strings.HasPrefix(digits, "7") {
		return "", fmt.Errorf("invalid phone number format after normalization: %s", digits)
	}

	return "+" + digits, nil
}

func printTeam(submission SubmissionResponse, mode *string) string {
	teamName := submission.TeamName
	captainPhone := ""
	if mode != nil {
		switch *mode {
		case tgbotapi.ModeHTML:
			teamName = "<b>" + teamName + "</b>"
		case "WHATSAPP":
			teamName = "*" + teamName + "*"
			captainPhone = submission.PhoneNumber
		case tgbotapi.ModeMarkdown, tgbotapi.ModeMarkdownV2:
			teamName = "**" + teamName + "**"
		}
	}
	msg := fmt.Sprintf(
		"Team \"%s\" registered!\n\nSquad:\n1. %s %s (%d) %s\n2. %s %s (%d)\n3. %s %s (%d)\n4. %s %s (%d)\n5. %s %s (%d)",
		teamName, strings.Trim(submission.CaptainLastName, " "), strings.Trim(submission.CaptainFirstName, " "), submission.CaptainGradYear, captainPhone,
		strings.Trim(submission.Member2LastName, " "), strings.Trim(submission.Member2FirstName, " "), submission.Member2GradYear,
		strings.Trim(submission.Member3LastName, " "), strings.Trim(submission.Member3FirstName, " "), submission.Member3GradYear,
		strings.Trim(submission.Member4LastName, " "), strings.Trim(submission.Member4FirstName, " "), submission.Member4GradYear,
		strings.Trim(submission.Member5LastName, " "), strings.Trim(submission.Member5FirstName, " "), submission.Member5GradYear,
	)

	if submission.ReserveLastName != nil && submission.ReserveFirstName != nil && submission.ReserveGradYear != nil {
		msg += fmt.Sprintf("\n6. %s %s (%d)", *submission.ReserveLastName, *submission.ReserveFirstName, *submission.ReserveGradYear)
	}

	return msg
}
