package main

import (
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
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"gopkg.in/gomail.v2"
)

type Template struct {
	templates *template.Template
}

// isDebug controls verbose logging via env var DEBUG_LOG=1
func isDebug() bool { return os.Getenv("DEBUG_LOG") == "1" }

// newSMTPDialer creates a gomail dialer using env configuration.
// Supports STARTTLS on port 587 (default) and SSL on port 465 when SMTP_PORT=465.
func newSMTPDialer() *gomail.Dialer {
	host := os.Getenv("SMTP_HOST")
	user := os.Getenv("SMTP_USERNAME")
	pass := os.Getenv("SMTP_PASSWORD")
	portStr := os.Getenv("SMTP_PORT")
	if portStr == "" {
		portStr = "587"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 587
	}
	d := gomail.NewDialer(host, port, user, pass)
	// If using implicit SSL (465), enable SSL mode
	if port == 465 {
		d.SSL = true
	}
	return d
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
		"templates/coffee.html",
		"templates/crossfit.html",
		"templates/howitworks.html",
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
	})

	t := &Template{
		templates: template.Must(tmpl.ParseFiles(mainTemplates...)),
	}
	e.Renderer = t

	// Static assets
	e.Static("/assets", "assets")

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
	e.GET("/coffee", func(c echo.Context) error {
		return c.Render(http.StatusOK, "coffee.html", getTemplateData("Coffee Service"))
	})
	e.GET("/crossfit", func(c echo.Context) error {
		return c.Render(http.StatusOK, "crossfit.html", getTemplateData("CrossFit Training"))
	})
	e.GET("/howitworks", func(c echo.Context) error {
		return c.Render(http.StatusOK, "howitworks.html", getTemplateData("How It Works"))
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
	e.File("/blog", "static/blog/index.html")
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

		m := gomail.NewMessage()
		m.SetHeader("From", os.Getenv("SMTP_FROM_EMAIL"))
		m.SetHeader("To", os.Getenv("SMTP_TO_EMAIL"))
		m.SetHeader("Subject", "Still Waters Contact Form")
		m.SetBody("text/html", "Email: "+email+"<br>Message: "+message)

		d := newSMTPDialer()

		if err := d.DialAndSend(m); err != nil {
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

			// Setup SMTP
			m := gomail.NewMessage()
			m.SetHeader("From", os.Getenv("SMTP_FROM_EMAIL")) // My verified SES email
			m.SetHeader("To", os.Getenv("SMTP_TO_EMAIL"))
			m.SetHeader("Subject", "Still Waters Contact Form")
			m.SetBody("text/html", "Email: "+email+"<br>Message: "+message)

			d := newSMTPDialer()

			// Send the email
			if err := d.DialAndSend(m); err != nil {
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
	prop := c.FormValue("prop")

	// Block the dates

	// err := updateCalendar(prop, start, end, "false")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }

	return c.Redirect(http.StatusFound, "/pricing?start="+start+"&end="+end+"&total="+total+"&prop="+prop)
}

func bookingGet(c echo.Context) error {

	if err := c.Request().ParseForm(); err != nil {
		return err
	}

	start := c.FormValue("start-date")
	end := c.FormValue("end-date")
	total := c.FormValue("total-price")
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

	qs := "start=" + start + "&end=" + end + "&total=" + total + "&prop=" + prop + "&rs=" + rs
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

			// Setup SMTP
			m := gomail.NewMessage()
			m.SetHeader("From", os.Getenv("SMTP_FROM_EMAIL"))
			m.SetHeader("To", os.Getenv("SMTP_TO_EMAIL"))
			m.SetHeader("Subject", "Still Waters Booking Request!")
			m.SetBody("text/html", "Cabin: "+cabin+"<br>Email: "+email+"<br>Message: "+message+"<br>Checkin: "+checkin+"<br>Checkout: "+checkout+"<br>Package: "+pkg+"<br>Price: "+price)

			d := newSMTPDialer()

			// Send the email
			if err := d.DialAndSend(m); err != nil {
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
	templatePath := "templates/blog_template.html"

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
	var postData map[string]string
	err = json.Unmarshal(jsonContent, &postData)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error parsing blog post data")
	}

	// Read the template file
	templateContent, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error loading template")
	}

	// Replace placeholders with blog content
	htmlContent := string(templateContent)
	for key, value := range postData {
		placeholder := fmt.Sprintf("{{ %s }}", key)
		htmlContent = strings.ReplaceAll(htmlContent, placeholder, value)
	}

	return c.HTML(http.StatusOK, htmlContent)
}

// getReviews fetches 20 latest 5-star reviews for a property via Hospitable API
// Requires env: HOSPITABLE_API_KEY and property IDs (same as calendar) mapped by prop code
// getReviews removed - static JSON is used for rendering reviews
