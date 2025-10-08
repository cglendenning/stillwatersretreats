package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/paymentintent"
)

type Template struct {
	templates *template.Template
}

type AnalyticsEvent struct {
	Timestamp string `json:"timestamp"`
	EventType string `json:"event_type"`
	Page      string `json:"page,omitempty"`
	Cabin     string `json:"cabin,omitempty"`
	Amount    string `json:"amount,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

type DashboardMetrics struct {
	TotalPageViews     int              `json:"total_page_views"`
	UniqueVisitors     int              `json:"unique_visitors"`
	CabinSelections    map[string]int   `json:"cabin_selections"`
	BookingAttempts    int              `json:"booking_attempts"`
	PaymentIntents     int              `json:"payment_intents"`
	CheckoutSessions   int              `json:"checkout_sessions"`
	ContactFormSubmits int              `json:"contact_form_submits"`
	LastUpdated        string           `json:"last_updated"`
	RecentEvents       []AnalyticsEvent `json:"recent_events"`

	// Time-based data
	PageViewsByHour  map[string]int `json:"page_views_by_hour"`
	PageViewsByDay   map[string]int `json:"page_views_by_day"`
	ConversionsByDay map[string]int `json:"conversions_by_day"`

	// Route tracking
	PageViewsByRoute map[string]int `json:"page_views_by_route"`

	// Comparison metrics
	PrevPeriodPageViews   int `json:"prev_period_page_views"`
	PrevPeriodVisitors    int `json:"prev_period_visitors"`
	PrevPeriodConversions int `json:"prev_period_conversions"`

	// Funnel metrics
	FunnelSteps map[string]int `json:"funnel_steps"`
}

// isDebug controls verbose logging via env var DEBUG_LOG=1
func isDebug() bool { return os.Getenv("DEBUG_LOG") == "1" }

// logAnalyticsEvent appends an analytics event to the analytics.json file
func logAnalyticsEvent(eventType, page, cabin, amount, userAgent string) error {
	event := AnalyticsEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		EventType: eventType,
		Page:      page,
		Cabin:     cabin,
		Amount:    amount,
		UserAgent: userAgent,
	}

	// Read existing events
	var events []AnalyticsEvent
	data, err := ioutil.ReadFile("assets/data/analytics.json")
	if err == nil {
		json.Unmarshal(data, &events)
	}

	// Append new event
	events = append(events, event)

	// Keep only last 1000 events to prevent file bloat
	if len(events) > 1000 {
		events = events[len(events)-1000:]
	}

	// Write back
	jsonData, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile("assets/data/analytics.json", jsonData, 0644)
}

// getDashboardMetrics calculates metrics from analytics.json with optional time period
func getDashboardMetrics() (DashboardMetrics, error) {
	return getDashboardMetricsForPeriod(0) // 0 = all time
}

// getDashboardMetricsForPeriod calculates metrics for a specific time period
// period: 0=all time, 1=24h, 7=7 days, 30=30 days
func getDashboardMetricsForPeriod(period int) (DashboardMetrics, error) {
	metrics := DashboardMetrics{
		CabinSelections:  make(map[string]int),
		PageViewsByHour:  make(map[string]int),
		PageViewsByDay:   make(map[string]int),
		ConversionsByDay: make(map[string]int),
		PageViewsByRoute: make(map[string]int),
		FunnelSteps:      make(map[string]int),
		LastUpdated:      time.Now().Format("2006-01-02 15:04:05"),
	}

	data, err := ioutil.ReadFile("assets/data/analytics.json")
	if err != nil {
		return metrics, err
	}

	var events []AnalyticsEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return metrics, err
	}

	now := time.Now()
	var cutoffTime time.Time
	if period > 0 {
		cutoffTime = now.AddDate(0, 0, -period)
	}

	// Previous period cutoff for comparison
	var prevPeriodStart, prevPeriodEnd time.Time
	if period > 0 {
		prevPeriodEnd = cutoffTime
		prevPeriodStart = cutoffTime.AddDate(0, 0, -period)
	}

	uniqueIPs := make(map[string]bool)
	prevUniqueIPs := make(map[string]bool)

	for _, event := range events {
		eventTime, err := time.Parse(time.RFC3339, event.Timestamp)
		if err != nil {
			continue
		}

		// Skip if outside time period
		if period > 0 && eventTime.Before(cutoffTime) {
			// Check if in previous period for comparison
			if eventTime.After(prevPeriodStart) && eventTime.Before(prevPeriodEnd) {
				switch event.EventType {
				case "page_view":
					metrics.PrevPeriodPageViews++
					prevUniqueIPs[event.UserAgent] = true
				case "checkout_session":
					metrics.PrevPeriodConversions++
				}
			}
			continue
		}

		// Current period metrics
		switch event.EventType {
		case "page_view":
			metrics.TotalPageViews++
			uniqueIPs[event.UserAgent] = true

			// Hour bucket (for 24h view)
			hourKey := eventTime.Format("2006-01-02 15:00")
			metrics.PageViewsByHour[hourKey]++

			// Day bucket
			dayKey := eventTime.Format("2006-01-02")
			metrics.PageViewsByDay[dayKey]++

			// Route tracking
			route := event.Page
			if route == "" {
				route = "/"
			}
			metrics.PageViewsByRoute[route]++

			// Funnel: Step 1
			metrics.FunnelSteps["1_page_view"]++

		case "cabin_selection":
			metrics.CabinSelections[event.Cabin]++
			// Funnel: Step 2
			metrics.FunnelSteps["2_cabin_selection"]++

		case "booking_attempt":
			metrics.BookingAttempts++
			// Funnel: Step 3
			metrics.FunnelSteps["3_booking_attempt"]++

		case "payment_intent":
			metrics.PaymentIntents++

		case "checkout_session":
			metrics.CheckoutSessions++
			dayKey := eventTime.Format("2006-01-02")
			metrics.ConversionsByDay[dayKey]++
			// Funnel: Step 4
			metrics.FunnelSteps["4_checkout"]++

		case "contact_form":
			metrics.ContactFormSubmits++
		}
	}

	metrics.UniqueVisitors = len(uniqueIPs)
	metrics.PrevPeriodVisitors = len(prevUniqueIPs)

	// Get last 20 events in period
	var recentEvents []AnalyticsEvent
	for i := len(events) - 1; i >= 0 && len(recentEvents) < 20; i-- {
		eventTime, err := time.Parse(time.RFC3339, events[i].Timestamp)
		if err != nil {
			continue
		}
		if period == 0 || eventTime.After(cutoffTime) {
			recentEvents = append(recentEvents, events[i])
		}
	}
	metrics.RecentEvents = recentEvents

	return metrics, nil
}

// newSESClient creates an AWS SES v2 client
// When running on AWS (EC2, ECS, Lambda, etc.), it automatically uses the attached IAM role
// For local development, you can provide AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY in .env
func newSESClient(ctx context.Context) (*sesv2.Client, error) {
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-east-1" // Default region
	}

	// Check if we have explicit credentials (for local dev)
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	var cfg aws.Config
	var err error

	if awsAccessKey != "" && awsSecretKey != "" {
		// Use explicit credentials (local development)
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsRegion),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				awsAccessKey,
				awsSecretKey,
				"",
			)),
		)
	} else {
		// Use IAM role (production on AWS)
		// This automatically picks up credentials from EC2 instance metadata, ECS task role, etc.
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsRegion),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	return sesv2.NewFromConfig(cfg), nil
}

// sendEmailViaSES sends an email using AWS SES v2 API
// replyToEmail is optional - if provided, replies will go to this address
func sendEmailViaSES(ctx context.Context, fromEmail, toEmail, subject, htmlBody, replyToEmail string) error {
	client, err := newSESClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create SES client: %w", err)
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(fromEmail),
		Destination: &types.Destination{
			ToAddresses: []string{toEmail},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data: aws.String(subject),
				},
				Body: &types.Body{
					Html: &types.Content{
						Data: aws.String(htmlBody),
					},
				},
			},
		},
	}

	// Add Reply-To header if provided
	if replyToEmail != "" {
		input.ReplyToAddresses = []string{replyToEmail}
	}

	_, err = client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send email via SES: %w", err)
	}

	return nil
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func getTemplateData(title string) map[string]interface{} {
	return map[string]interface{}{
		"Title":              title,
		"FIREBASE_API_KEY":   os.Getenv("FIREBASE_API_KEY"),
		"GOOGLE_TAG_ID":      os.Getenv("GOOGLE_TAG_ID"),
		"YOUTUBE_VIDEO_ID":   os.Getenv("YOUTUBE_VIDEO_ID"),
		"RECAPTCHA_SITE_KEY": os.Getenv("RECAPTCHA_SITE_KEY"),
	}
}

func convertDateFormat(dateStr string) string {
	// Parse the input date (assuming YYYY-MM-DD format)
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		// If parsing fails, return the original string
		return dateStr
	}
	// Format as MM/DD/YYYY
	return t.Format("01/02/2006")
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("[DEBUG] No .env file found or error loading .env:", err)
	} else {
		fmt.Println("[DEBUG] .env file loaded successfully")
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Template renderer - parse main templates with custom functions
	mainTemplates := []string{
		"templates/nav.html",
		"templates/footer.html",
		"templates/home.html",
		"templates/cabins.html",
		"templates/bearviewcove.html",
		"templates/victoriapines.html",
		"templates/activities.html",
		"templates/coaching.html",
		"templates/coaching-detail.html",
		"templates/coachcraig.html",
		"templates/coffee.html",
		"templates/crossfit.html",
		"templates/howitworks.html",
		"templates/survey.html",
		"templates/pricing.html",
		"templates/packages.html",
		"templates/calendar.html",
		"templates/contact.html",
		"templates/success.html",
		"templates/cancel.html",
		"templates/review.html",
		"templates/booking.html",
		"templates/email-sent.html",
		"templates/book-bearview.html",
		"templates/blog_list.html",
		"templates/blog_template.html",
		"templates/dashboard.html",
	}

	// Create template with custom functions
	tmpl := template.New("").Funcs(template.FuncMap{
		"FIREBASE_API_KEY":   func() string { return os.Getenv("FIREBASE_API_KEY") },
		"GOOGLE_TAG_ID":      func() string { return os.Getenv("GOOGLE_TAG_ID") },
		"YOUTUBE_VIDEO_ID":   func() string { return os.Getenv("YOUTUBE_VIDEO_ID") },
		"RECAPTCHA_SITE_KEY": func() string { return os.Getenv("RECAPTCHA_SITE_KEY") },
		"meta_description": func() string {
			return "Still Waters Retreat - Quiet mountain cabins for deep thinking and creative work"
		},
		"add":     func(a, b int) int { return a + b },
		"float64": func(i int) float64 { return float64(i) },
		"divf": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mulf": func(a, b float64) float64 { return a * b },
		"toJSON": func(v interface{}) string {
			bytes, _ := json.Marshal(v)
			return string(bytes)
		},
	})

	t := &Template{
		templates: template.Must(tmpl.ParseFiles(mainTemplates...)),
	}
	e.Renderer = t

	// Static assets
	e.Static("/assets", "assets")
	e.Static("/blog-data", "blog")

	// Routes
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "home.html", getTemplateData("Still Waters Retreat"))
	})
	e.File("/sitemap.xml", "sitemap.xml")
	e.File("/googleb900bf2db224a84c.html", "googleb900bf2db224a84c.html")
	e.File("/robots.txt", "robots.txt")
	e.GET("/cabins", func(c echo.Context) error {
		return c.Render(http.StatusOK, "cabins.html", getTemplateData("Cabins"))
	})
	e.GET("/bearviewcove", func(c echo.Context) error {
		return c.Render(http.StatusOK, "bearviewcove.html", getTemplateData("Bear View Cove"))
	})
	e.GET("/victoriapines", func(c echo.Context) error {
		return c.Render(http.StatusOK, "victoriapines.html", getTemplateData("Victoria Pines"))
	})
	e.GET("/activities", func(c echo.Context) error {
		return c.Render(http.StatusOK, "activities.html", getTemplateData("Activities"))
	})
	e.GET("/coaching", func(c echo.Context) error {
		return c.Render(http.StatusOK, "coaching.html", getTemplateData("Coaching with Craig"))
	})
	e.GET("/coaching-detail", func(c echo.Context) error {
		return c.Render(http.StatusOK, "coaching-detail.html", getTemplateData("Coaching Options"))
	})
	e.GET("/coachcraig", func(c echo.Context) error {
		return c.Render(http.StatusOK, "coachcraig.html", getTemplateData("Coach Craig"))
	})
	e.GET("/coffee", func(c echo.Context) error {
		return c.Render(http.StatusOK, "coffee.html", getTemplateData("Coffee Service"))
	})
	e.GET("/crossfit", func(c echo.Context) error {
		return c.Render(http.StatusOK, "crossfit.html", getTemplateData("CrossFit Training"))
	})
	e.GET("/howitworks", func(c echo.Context) error {
		return c.Render(http.StatusOK, "howitworks.html", getTemplateData("How It Works"))
	})
	e.GET("/survey", func(c echo.Context) error {
		return c.Render(http.StatusOK, "survey.html", getTemplateData("Boxed-In Builder Survey"))
	})
	e.GET("/pricing", func(c echo.Context) error {
		return c.Render(http.StatusOK, "pricing.html", getTemplateData("Pricing"))
	})
	e.GET("/packages", func(c echo.Context) error {
		return c.Render(http.StatusOK, "packages.html", getTemplateData("Packages"))
	})
	e.GET("/calendar", func(c echo.Context) error {
		return c.Render(http.StatusOK, "calendar.html", getTemplateData("Calendar"))
	})
	e.GET("/contact", func(c echo.Context) error {
		return c.Render(http.StatusOK, "contact.html", getTemplateData("Book Your Retreat"))
	})
	e.GET("/success", func(c echo.Context) error {
		return c.Render(http.StatusOK, "success.html", getTemplateData("Thank You"))
	})
	e.GET("/cancel", func(c echo.Context) error {
		return c.Render(http.StatusOK, "cancel.html", getTemplateData("Thank You"))
	})
	e.GET("/review", func(c echo.Context) error {
		return c.Render(http.StatusOK, "review.html", getTemplateData("Review Your Booking"))
	})
	e.GET("/booking", func(c echo.Context) error {
		return c.Render(http.StatusOK, "booking.html", getTemplateData("Booking"))
	})
	e.GET("/email-sent", func(c echo.Context) error {
		return c.Render(http.StatusOK, "email-sent.html", getTemplateData("Email Sent"))
	})
	e.GET("/book-bearview", func(c echo.Context) error {
		return c.Render(http.StatusOK, "book-bearview.html", getTemplateData("Book Bear View"))
	})
	e.File("/greenpyramid/setup-video", "templates/greenpyramid/setup-video.html")
	e.File("/greenpyramid/paywall-video", "templates/greenpyramid/paywall-video.html")
	e.File("/greenpyramid/sms-opt-in", "templates/greenpyramid/sms-opt-in.html")
	e.Static("/static", "static")

	// Blog routes
	e.GET("/blog", func(c echo.Context) error {
		return c.Render(http.StatusOK, "blog_list.html", getTemplateData("Blog - Still Waters Retreats"))
	})
	e.GET("/blog/:post", serveBlogPost)

	// handling data
	e.POST("/create-checkout-session", createCheckoutSessionHandler)
	e.POST("/create-payment-intent", createPaymentIntentHandler)
	e.GET("/calendar-data", getCalendar)
	e.GET("/property-data", getProperties)
	e.GET("/dates-chosen", selectedDates)
	e.GET("/booking-get", bookingGet)
	e.GET("/api/images/:cabin", getImages)
	e.POST("/send-email", sendEmailHandler)
	e.POST("/send-booking", sendBookingHandler)

	// Analytics tracking
	e.POST("/api/track", trackEventHandler)
	e.GET("/api/metrics", metricsAPIHandler)

	// Dashboard (password protected)
	e.GET("/dashboard", dashboardHandler)
	e.POST("/dashboard-login", dashboardLoginHandler)

	// Only enable HTTPS redirect in production (not localhost)
	if os.Getenv("ENVIRONMENT") != "local" {
		e.Pre(middleware.HTTPSRedirect())
	}
	e.Logger.Fatal(e.Start(":8080"))
}

func createCheckoutSessionHandler(c echo.Context) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	var requestData struct {
		Amount           int64  `json:"amount"`
		ImageURL         string `json:"image_url"`
		Cabin            string `json:"cabin"`
		StartDate        string `json:"start_date"`
		EndDate          string `json:"end_date"`
		RetreatStructure string `json:"retreat_structure"`
		Massage          string `json:"massage"`
		Meditation       string `json:"meditation"`
		Hike             string `json:"hike"`
	}

	if err := c.Bind(&requestData); err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}

	// Debug: Print received data
	fmt.Printf("Received booking data: %+v\n", requestData)

	// Convert cabin code to proper name
	cabinName := requestData.Cabin
	switch requestData.Cabin {
	case "bvc":
		cabinName = "Bear View Cove"
	case "vp":
		cabinName = "Victoria Pines"
	}

	// Convert dates to MM/DD/YYYY format
	startDate := convertDateFormat(requestData.StartDate)
	endDate := convertDateFormat(requestData.EndDate)

	// Build detailed description with better formatting
	description := fmt.Sprintf("%s Cabin\nCheck in: %s\nCheck out: %s",
		cabinName, startDate, endDate)

	// Add retreat structure if it's not empty and not "my-own"
	if requestData.RetreatStructure != "" && requestData.RetreatStructure != "my-own" {
		description += fmt.Sprintf(" | Structure: %s", requestData.RetreatStructure)
	}

	// Filter out "no" entries and build add-ons list
	addOns := []string{}
	if requestData.Massage != "" && requestData.Massage != "no" && requestData.Massage != "None" {
		addOns = append(addOns, requestData.Massage)
	}
	if requestData.Meditation != "" && requestData.Meditation != "no" && requestData.Meditation != "None" {
		addOns = append(addOns, requestData.Meditation)
	}
	if requestData.Hike != "" && requestData.Hike != "no" && requestData.Hike != "None" {
		addOns = append(addOns, requestData.Hike)
	}

	// Only add add-ons section if there are actual add-ons
	if len(addOns) > 0 {
		description += fmt.Sprintf("\nAdd-ons: %s", strings.Join(addOns, ", "))
	}

	// Build dynamic base URL from current request (works locally and in production)
	scheme := c.Scheme()
	if xf := c.Request().Header.Get("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	host := c.Request().Host
	baseURL := scheme + "://" + host

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			&stripe.CheckoutSessionLineItemParams{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(description),
						Description: stripe.String(description),
						Images:      stripe.StringSlice([]string{requestData.ImageURL}),
					},
					UnitAmount: stripe.Int64(requestData.Amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(baseURL + "/success"),
		CancelURL:  stripe.String(baseURL + "/cancel"),
	}

	s, err := session.New(params)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	// Track checkout session creation
	amount := fmt.Sprintf("$%.2f", float64(requestData.Amount)/100)
	logAnalyticsEvent("checkout_session", "", cabinName, amount, c.Request().Header.Get("User-Agent"))

	return c.JSON(http.StatusOK, map[string]string{"sessionId": s.ID})
}

func createPaymentIntentHandler(c echo.Context) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	var requestData struct {
		Amount           int64  `json:"amount"`
		ImageURL         string `json:"image_url"`
		Cabin            string `json:"cabin"`
		StartDate        string `json:"start_date"`
		EndDate          string `json:"end_date"`
		RetreatStructure string `json:"retreat_structure"`
		Massage          string `json:"massage"`
		Meditation       string `json:"meditation"`
		Hike             string `json:"hike"`
	}

	if err := c.Bind(&requestData); err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}

	// Convert cabin code to proper name
	cabinName := requestData.Cabin
	switch requestData.Cabin {
	case "bvc":
		cabinName = "Bear View Cove"
	case "vp":
		cabinName = "Victoria Pines"
	}

	// Convert dates to MM/DD/YYYY format
	startDate := convertDateFormat(requestData.StartDate)
	endDate := convertDateFormat(requestData.EndDate)

	// Build description
	description := fmt.Sprintf("%s Cabin (%s to %s)", cabinName, startDate, endDate)

	// Create payment intent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(requestData.Amount),
		Currency: stripe.String("usd"),
		Metadata: map[string]string{
			"cabin":             requestData.Cabin,
			"start_date":        requestData.StartDate,
			"end_date":          requestData.EndDate,
			"retreat_structure": requestData.RetreatStructure,
			"massage":           requestData.Massage,
			"meditation":        requestData.Meditation,
			"hike":              requestData.Hike,
		},
		Description: stripe.String(description),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	// Track payment intent creation
	amount := fmt.Sprintf("$%.2f", float64(requestData.Amount)/100)
	logAnalyticsEvent("payment_intent", "", cabinName, amount, c.Request().Header.Get("User-Agent"))

	return c.JSON(http.StatusOK, map[string]string{"client_secret": pi.ClientSecret})
}

func getImages(c echo.Context) error {
	cabin := c.Param("cabin")
	dir := filepath.Join("assets/photo", cabin)

	// Open the directory
	f, err := os.Open(dir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not open directory"})
	}
	defer f.Close()

	// Read the files in the directory
	files, err := f.Readdir(-1)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not read directory"})
	}

	var imageUrls []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jpg") {
			imageUrls = append(imageUrls, "/assets/photo/"+cabin+"/"+file.Name())
		}
	}

	return c.JSON(http.StatusOK, imageUrls)
}

func sendEmailHandler(c echo.Context) error {
	// Allow skipping reCAPTCHA locally via env flag
	if os.Getenv("SKIP_RECAPTCHA") == "1" {
		// Parse form data
		email := c.FormValue("email")
		message := c.FormValue("message")

		fromEmail := os.Getenv("SES_FROM_EMAIL")
		toEmail := os.Getenv("SES_TO_EMAIL")
		subject := "Still Waters Contact Form"
		htmlBody := "Email: " + email + "<br>Message: " + message

		// Use customer's email as Reply-To so you can easily reply to them
		if err := sendEmailViaSES(c.Request().Context(), fromEmail, toEmail, subject, htmlBody, email); err != nil {
			return c.String(http.StatusInternalServerError, "Failed to send email: "+err.Error())
		}

		return c.Redirect(http.StatusFound, "/email-sent")
	}

	recaptchaResponse := c.FormValue("g-recaptcha-response")

	projectID := os.Getenv("RECAPTCHA_PROJECT_ID")
	apiKey := os.Getenv("RECAPTCHA_API_KEY")
	recaptchaSiteKey := os.Getenv("RECAPTCHA_SITE_KEY")

	verifyURL := fmt.Sprintf("https://recaptchaenterprise.googleapis.com/v1/projects/%s/assessments?key=%s", projectID, apiKey)
	postData := map[string]interface{}{
		"event": map[string]string{
			"token":   recaptchaResponse,
			"siteKey": recaptchaSiteKey,
		},
	}
	postBody, _ := json.Marshal(postData)

	// Make the HTTP POST request
	req, _ := http.NewRequest("POST", verifyURL, strings.NewReader(string(postBody)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to verify CAPTCHA: "+err.Error())
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	// Check the assessment score and other details
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if success, ok := result["tokenProperties"].(map[string]interface{})["valid"].(bool); ok && success {

		if score, found := result["riskAnalysis"].(map[string]interface{})["score"].(float64); found && score > 0.5 {

			// Parse form data
			email := c.FormValue("email")
			message := c.FormValue("message")

			fromEmail := os.Getenv("SES_FROM_EMAIL")
			toEmail := os.Getenv("SES_TO_EMAIL")
			subject := "Still Waters Contact Form"
			htmlBody := "Email: " + email + "<br>Message: " + message

			// Send the email via SES with customer's email as Reply-To
			if err := sendEmailViaSES(c.Request().Context(), fromEmail, toEmail, subject, htmlBody, email); err != nil {
				return c.String(http.StatusInternalServerError, "Failed to send email: "+err.Error())
			}

			return c.Redirect(http.StatusFound, "/email-sent")
		}
	}
	// I could parameterize contact.html and send an indication that recaptcha failed.
	return c.Redirect(http.StatusFound, "/contact")
}

func getProperties(c echo.Context) error {

	url := "https://public.api.hospitable.com/v2/properties"
	// Set up the request to the Hospitable API
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	apiKey := os.Getenv("HOSPITABLE_API_KEY")

	// Include your API key in the header
	req.Header.Add("Authorization", "Bearer "+apiKey)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	defer resp.Body.Close()

	// Read the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSONBlob(http.StatusOK, body)
}

func getCalendar(c echo.Context) error {

	today := time.Now()
	beginDate := today.Format("2006-01-02")

	sixMonthsFromNow := today.AddDate(0, 6, 0)
	endDate := sixMonthsFromNow.Format("2006-01-02")

	propId := ""
	if c.QueryParam("prop") == "vp" {
		propId = "833d93ba-7d6f-4d02-b19b-26da46f72e37"
	} else if c.QueryParam("prop") == "bvc" {
		propId = "dd5e28be-2406-4624-908d-30313972781d"
	} else {
		if isDebug() {
			fmt.Println("[DEBUG] Invalid or missing prop param:", c.QueryParam("prop"))
		}
		return c.Redirect(http.StatusFound, "/contact")
	}

	url := "https://public.api.hospitable.com/v2/properties/" + propId + "/calendar?start_date=" + beginDate + "&end_date=" + endDate
	if isDebug() {
		fmt.Println("[DEBUG] Calendar API URL:", url)
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if isDebug() {
			fmt.Println("[DEBUG] Error creating request:", err)
		}
		return c.String(http.StatusInternalServerError, err.Error())
	}

	apiKey := os.Getenv("HOSPITABLE_API_KEY")
	if isDebug() {
		fmt.Println("[DEBUG] HOSPITABLE_API_KEY in getCalendar:", apiKey)
	}

	req.Header.Add("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		if isDebug() {
			fmt.Println("[DEBUG] Error making request:", err)
		}
		return c.String(http.StatusInternalServerError, err.Error())
	}
	defer resp.Body.Close()

	if isDebug() {
		fmt.Println("[DEBUG] Calendar API response status:", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if isDebug() {
			fmt.Println("[DEBUG] Error reading response body:", err)
		}
		return c.String(http.StatusInternalServerError, err.Error())
	}

	if isDebug() {
		fmt.Println("[DEBUG] Calendar API response body:", string(body))
	}

	return c.JSONBlob(http.StatusOK, body)
}

func updateCalendar(prop, start, end, available string) error {

	propId := ""
	if prop == "vp" {
		propId = "833d93ba-7d6f-4d02-b19b-26da46f72e37"
	} else if prop == "bvc" {
		propId = "dd5e28be-2406-4624-908d-30313972781d"
	} else {
		return errors.New("Property not found.")
	}

	url := "https://public.api.hospitable.com/v2/properties/" + propId + "/calendar"

	// TODO: Make this block temporary, until you have money in the bank.

	startDate, err := time.Parse("2006-01-02", start)
	if err != nil {
		return errors.New("Invalid start date format")
	}

	endDate, err := time.Parse("2006-01-02", end)
	if err != nil {
		return errors.New("Invalid end date format")
	}

	if endDate.Before(startDate) {
		return errors.New("End date must be after start date")
	}

	var dates string
	for date := startDate; date.Before(endDate); date = date.AddDate(0, 0, 1) {
		dates += "{\"date\":\"" + date.Format("2006-01-02") + "\",\"available\":" + available + "},"
	}

	// strip off the last comma
	dates = dates[:len(dates)-1]

	payload := strings.NewReader("{\"dates\":[" + dates + "]}")

	// Set up the request to the Hospitable API
	client := &http.Client{}

	req, err := http.NewRequest("PUT", url, payload)
	if err != nil {
		return err
	}

	apiKey := os.Getenv("HOSPITABLE_API_KEY")

	// Include your API key in the header
	req.Header.Add("accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+apiKey)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func selectedDates(c echo.Context) error {

	start := c.FormValue("start-date")
	end := c.FormValue("end-date")
	total := c.FormValue("total-price")
	nightlyPrices := c.FormValue("nightly-prices")
	prop := c.FormValue("prop")

	// Block the dates

	// err := updateCalendar(prop, start, end, "false")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }

	return c.Redirect(http.StatusFound, "/pricing?start="+start+"&end="+end+"&total="+total+"&nights="+url.QueryEscape(nightlyPrices)+"&prop="+prop)
}

func bookingGet(c echo.Context) error {

	if err := c.Request().ParseForm(); err != nil {
		return err
	}

	start := c.FormValue("start-date")
	end := c.FormValue("end-date")
	total := c.FormValue("total-price")
	nightlyPrices := c.FormValue("nights")
	prop := c.FormValue("prop")
	rs := c.FormValue("retreat-structure")

	// New coaching add-on from pricing page
	coachingOption := c.FormValue("coaching-option")
	coachingLabel := ""
	switch coachingOption {
	case "virtual-live":
		coachingLabel = "Virtual live coaching ($499)"
	case "in-person-4hr":
		coachingLabel = "In-person 1:1 coaching (4 hours) ($997)"
	}

	qs := "start=" + start + "&end=" + end + "&total=" + total + "&nights=" + nightlyPrices + "&prop=" + prop + "&rs=" + rs
	if coachingLabel != "" {
		qs += "&coaching=" + url.QueryEscape(coachingLabel)
	}
	return c.Redirect(http.StatusFound, "/review?"+qs)
}

func sendBookingHandler(c echo.Context) error {
	recaptchaResponse := c.FormValue("g-recaptcha-response")

	projectID := os.Getenv("RECAPTCHA_PROJECT_ID")
	apiKey := os.Getenv("RECAPTCHA_API_KEY")
	recaptchaSiteKey := os.Getenv("RECAPTCHA_SITE_KEY")

	verifyURL := fmt.Sprintf("https://recaptchaenterprise.googleapis.com/v1/projects/%s/assessments?key=%s", projectID, apiKey)
	postData := map[string]interface{}{
		"event": map[string]string{
			"token":   recaptchaResponse,
			"siteKey": recaptchaSiteKey,
		},
	}
	postBody, _ := json.Marshal(postData)

	// Make the HTTP POST request
	req, _ := http.NewRequest("POST", verifyURL, strings.NewReader(string(postBody)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to verify CAPTCHA: "+err.Error())
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	// Check the assessment score and other details
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if success, ok := result["tokenProperties"].(map[string]interface{})["valid"].(bool); ok && success {

		if score, found := result["riskAnalysis"].(map[string]interface{})["score"].(float64); found && score > 0.5 {

			// Parse form data

			email := c.FormValue("email")
			message := c.FormValue("message")
			cabin := c.FormValue("cabin-send")
			checkin := c.FormValue("checkin-send")
			checkout := c.FormValue("checkout-send")
			pkg := c.FormValue("pkg-send")
			price := c.FormValue("price-send")

			fromEmail := os.Getenv("SES_FROM_EMAIL")
			toEmail := os.Getenv("SES_TO_EMAIL")
			subject := "Still Waters Booking Request!"
			htmlBody := "Cabin: " + cabin + "<br>Email: " + email + "<br>Message: " + message + "<br>Checkin: " + checkin + "<br>Checkout: " + checkout + "<br>Package: " + pkg + "<br>Price: " + price

			// Send the email via SES with customer's email as Reply-To
			if err := sendEmailViaSES(c.Request().Context(), fromEmail, toEmail, subject, htmlBody, email); err != nil {
				return c.String(http.StatusInternalServerError, "Failed to send email: "+err.Error())
			}

			return c.Redirect(http.StatusFound, "/email-sent")
		}
	}
	// I could parameterize contact.html and send an indication that recaptcha failed.
	return c.Redirect(http.StatusFound, "/contact")
}

func serveBlogPost(c echo.Context) error {
	postID := c.Param("post")
	jsonFilePath := fmt.Sprintf("blog/%s.json", postID)

	// Read the JSON content file
	jsonFile, err := os.Open(jsonFilePath)
	if err != nil {
		return c.String(http.StatusNotFound, "Blog post not found")
	}
	defer jsonFile.Close()

	jsonContent, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error reading blog post")
	}

	// Parse JSON content
	var postData map[string]interface{}
	err = json.Unmarshal(jsonContent, &postData)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error parsing blog post data")
	}

	// Add environment variables and other data
	postData["FIREBASE_API_KEY"] = os.Getenv("FIREBASE_API_KEY")
	postData["slug"] = postID

	// Map blog topics to relevant YouTube videos
	youtubeVideo := getRelevantYouTubeVideo(postID)
	postData["youtube_embed"] = youtubeVideo

	return c.Render(http.StatusOK, "blog_template.html", postData)
}

// getRelevantYouTubeVideo returns a unique YouTube video for each blog post
// Uses a consistent hash-based approach to ensure each blog gets a different video
func getRelevantYouTubeVideo(postSlug string) string {
	// All available video IDs from @GreenPyramid-mk5xp channel
	// Each blog post gets assigned a unique video based on its slug
	videoIDs := []string{
		"uoxlK2QFlLA", "GnoRXRUwuDw", "Lu0Wko81utw", "j5UIk1qXtyE", "piJOUzQbOBQ",
		"4KIxb3dX9I0", "4Kxkoj7vQx8", "7jfJS3UOzjw", "7tkrJpWbXYE", "B7DxkPs9rWE",
		"EXwB0vT-cpk", "G-UMqRXquPM", "HKdIMXQtR1k", "I6IiREAVH_w", "IFwErIJk2Nw",
		"ILnjIQpA_Hw", "JVCK464ApNM", "K-o6ChjZRoE", "OezQxzcwf7g", "Q0IwvaK6bvc",
		"QBj8w1XEM6U", "XrcOY4phhXI", "Zf5V2_3ZsA8", "bBZL4MDy0Xo", "cF-UnjXK0Zg",
		"huPIWBJimgM", "p2P0AkF3y_s", "qRs5AnC4yro", "uFAxfa5QnCc", "xTeOUzzb29I",
		// Repeat the list to ensure we have enough for 100+ blog posts
		"uoxlK2QFlLA", "GnoRXRUwuDw", "Lu0Wko81utw", "j5UIk1qXtyE", "piJOUzQbOBQ",
		"4KIxb3dX9I0", "4Kxkoj7vQx8", "7jfJS3UOzjw", "7tkrJpWbXYE", "B7DxkPs9rWE",
		"EXwB0vT-cpk", "G-UMqRXquPM", "HKdIMXQtR1k", "I6IiREAVH_w", "IFwErIJk2Nw",
		"ILnjIQpA_Hw", "JVCK464ApNM", "K-o6ChjZRoE", "OezQxzcwf7g", "Q0IwvaK6bvc",
		"QBj8w1XEM6U", "XrcOY4phhXI", "Zf5V2_3ZsA8", "bBZL4MDy0Xo", "cF-UnjXK0Zg",
		"huPIWBJimgM", "p2P0AkF3y_s", "qRs5AnC4yro", "uFAxfa5QnCc", "xTeOUzzb29I",
		"uoxlK2QFlLA", "GnoRXRUwuDw", "Lu0Wko81utw", "j5UIk1qXtyE", "piJOUzQbOBQ",
		"4KIxb3dX9I0", "4Kxkoj7vQx8", "7jfJS3UOzjw", "7tkrJpWbXYE", "B7DxkPs9rWE",
		"EXwB0vT-cpk", "G-UMqRXquPM", "HKdIMXQtR1k", "I6IiREAVH_w", "IFwErIJk2Nw",
		"ILnjIQpA_Hw", "JVCK464ApNM", "K-o6ChjZRoE", "OezQxzcwf7g", "Q0IwvaK6bvc",
		"QBj8w1XEM6U", "XrcOY4phhXI", "Zf5V2_3ZsA8", "bBZL4MDy0Xo", "cF-UnjXK0Zg",
		"huPIWBJimgM", "p2P0AkF3y_s", "qRs5AnC4yro", "uFAxfa5QnCc", "xTeOUzzb29I",
		"uoxlK2QFlLA", "GnoRXRUwuDw", "Lu0Wko81utw", "j5UIk1qXtyE", "piJOUzQbOBQ",
	}

	// Use a simple hash of the slug to consistently pick a video
	// This ensures each blog post always gets the same video, but different posts get different videos
	hash := 0
	for _, char := range postSlug {
		hash = (hash*31 + int(char)) % len(videoIDs)
	}

	videoID := videoIDs[hash]
	return fmt.Sprintf("https://www.youtube.com/embed/%s?rel=0&modestbranding=1", videoID)
}

// getReviews fetches 20 latest 5-star reviews for a property via Hospitable API
// Requires env: HOSPITABLE_API_KEY and property IDs (same as calendar) mapped by prop code
// getReviews removed - static JSON is used for rendering reviews

// trackEventHandler logs analytics events
func trackEventHandler(c echo.Context) error {
	var req struct {
		EventType string `json:"event_type"`
		Page      string `json:"page"`
		Cabin     string `json:"cabin"`
		Amount    string `json:"amount"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	userAgent := c.Request().Header.Get("User-Agent")
	if err := logAnalyticsEvent(req.EventType, req.Page, req.Cabin, req.Amount, userAgent); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to log event"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// metricsAPIHandler returns metrics for specified time period
func metricsAPIHandler(c echo.Context) error {
	// Check auth
	cookie, err := c.Cookie("dashboard_auth")
	if err != nil || cookie.Value != os.Getenv("DASHBOARD_PASSWORD") {
		expectedPassword := os.Getenv("DASHBOARD_PASSWORD")
		if expectedPassword == "" {
			expectedPassword = "stillwaters2024"
		}
		if cookie.Value != expectedPassword {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		}
	}

	period := 0 // default to all time
	periodParam := c.QueryParam("period")
	if periodParam == "1" {
		period = 1
	} else if periodParam == "7" {
		period = 7
	} else if periodParam == "30" {
		period = 30
	}

	metrics, err := getDashboardMetricsForPeriod(period)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get metrics"})
	}

	return c.JSON(http.StatusOK, metrics)
}

// dashboardHandler serves the dashboard or login page
func dashboardHandler(c echo.Context) error {
	// Check for session cookie
	cookie, err := c.Cookie("dashboard_auth")
	if err == nil && cookie.Value == os.Getenv("DASHBOARD_PASSWORD") {
		// Authenticated - show dashboard
		metrics, err := getDashboardMetrics()
		if err != nil {
			// If no analytics file yet, show empty metrics
			metrics = DashboardMetrics{
				CabinSelections: make(map[string]int),
				LastUpdated:     time.Now().Format("2006-01-02 15:04:05"),
			}
		}

		data := getTemplateData("Dashboard")
		data["Metrics"] = metrics
		return c.Render(http.StatusOK, "dashboard.html", data)
	}

	// Not authenticated - show login form
	return c.HTML(http.StatusOK, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard Login</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .login-container {
            background: white;
            padding: 2.5rem;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            width: 90%;
            max-width: 400px;
        }
        h1 { 
            color: #333;
            margin-bottom: 1.5rem;
            font-size: 1.8rem;
            text-align: center;
        }
        .form-group {
            margin-bottom: 1.5rem;
        }
        label {
            display: block;
            margin-bottom: 0.5rem;
            color: #555;
            font-weight: 500;
        }
        input[type="password"] {
            width: 100%;
            padding: 0.75rem;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 1rem;
            transition: border-color 0.3s;
        }
        input[type="password"]:focus {
            outline: none;
            border-color: #667eea;
        }
        button {
            width: 100%;
            padding: 0.75rem;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        button:active {
            transform: translateY(0);
        }
        .error {
            color: #e74c3c;
            font-size: 0.9rem;
            margin-top: 1rem;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <h1>🔐 Dashboard Login</h1>
        <form method="POST" action="/dashboard-login">
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required autofocus>
            </div>
            <button type="submit">Access Dashboard</button>
        </form>
    </div>
</body>
</html>
	`)
}

// dashboardLoginHandler processes login attempts
func dashboardLoginHandler(c echo.Context) error {
	password := c.FormValue("password")
	expectedPassword := os.Getenv("DASHBOARD_PASSWORD")

	if expectedPassword == "" {
		expectedPassword = "stillwaters2024" // Default password if not set
	}

	if password == expectedPassword {
		// Set session cookie
		cookie := new(http.Cookie)
		cookie.Name = "dashboard_auth"
		cookie.Value = password
		cookie.Path = "/"
		cookie.MaxAge = 86400 // 24 hours
		cookie.HttpOnly = true
		c.SetCookie(cookie)
		return c.Redirect(http.StatusFound, "/dashboard")
	}

	// Failed login - show login page with error
	return c.HTML(http.StatusUnauthorized, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard Login</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .login-container {
            background: white;
            padding: 2.5rem;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            width: 90%;
            max-width: 400px;
        }
        h1 { 
            color: #333;
            margin-bottom: 1.5rem;
            font-size: 1.8rem;
            text-align: center;
        }
        .form-group {
            margin-bottom: 1.5rem;
        }
        label {
            display: block;
            margin-bottom: 0.5rem;
            color: #555;
            font-weight: 500;
        }
        input[type="password"] {
            width: 100%;
            padding: 0.75rem;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 1rem;
            transition: border-color 0.3s;
        }
        input[type="password"]:focus {
            outline: none;
            border-color: #667eea;
        }
        button {
            width: 100%;
            padding: 0.75rem;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        button:active {
            transform: translateY(0);
        }
        .error {
            color: #e74c3c;
            font-size: 0.9rem;
            margin-top: 1rem;
            text-align: center;
            font-weight: 500;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <h1>🔐 Dashboard Login</h1>
        <form method="POST" action="/dashboard-login">
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required autofocus>
            </div>
            <button type="submit">Access Dashboard</button>
            <div class="error">❌ Incorrect password</div>
        </form>
    </div>
</body>
</html>
	`)
}
