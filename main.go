package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"gopkg.in/gomail.v2"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Static assets
	e.Static("/assets", "assets")

	// Routes
	e.File("/", "templates/home.html")
	e.File("/sitemap.xml", "sitemap.xml")
	e.File("/googleb900bf2db224a84c.html", "googleb900bf2db224a84c.html")
	e.File("/robots.txt", "robots.txt")
	e.File("/cabins", "templates/cabins.html")
	e.File("/howitworks", "templates/howitworks.html")
	e.File("/pricing", "templates/pricing.html")
	e.File("/packages", "templates/packages.html")
	e.File("/calendar", "templates/calendar.html")
	e.File("/contact", "templates/contact.html")
	e.File("/success", "templates/success.html")
	e.File("/cancel", "templates/cancel.html")
	e.File("/review", "templates/review.html")
	e.File("/booking", "templates/booking.html")
	e.File("/email-sent", "templates/email-sent.html")
	e.File("/book-bearview", "templates/book-bearview.html")
	e.File("/greenpyramid/setup-video", "templates/greenpyramid/setup-video.html")
	e.File("/greenpyramid/paywall-video", "templates/greenpyramid/paywall-video.html")
	e.Static("/static", "static")

	// Blog routes
	e.File("/blog", "static/blog/index.html")
	e.GET("/blog/:post", serveBlogPost)

	// handling data
	e.POST("/create-checkout-session", createCheckoutSessionHandler)
	e.GET("/calendar-data", getCalendar)
	e.GET("/property-data", getProperties)
	e.GET("/dates-chosen", selectedDates)
	e.GET("/booking-get", bookingGet)
	e.GET("/api/images/:cabin", getImages)
	e.POST("/send-email", sendEmailHandler)
	e.POST("/send-booking", sendBookingHandler)

	e.Pre(middleware.HTTPSRedirect())
	e.Logger.Fatal(e.Start(":8080"))
}

func createCheckoutSessionHandler(c echo.Context) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	var requestData struct {
		Amount   int64  `json:"amount"`
		ImageURL string `json:"image_url"`
	}

	if err := c.Bind(&requestData); err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			&stripe.CheckoutSessionLineItemParams{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:   stripe.String("Your Retreat Booking"),
						Images: stripe.StringSlice([]string{requestData.ImageURL}),
					},
					UnitAmount: stripe.Int64(requestData.Amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String("https://stillwatersretreats.com/success"),
		CancelURL:  stripe.String("https://stillwatersretreats.com/cancel"),
	}

	s, err := session.New(params)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"sessionId": s.ID})
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

			d := gomail.NewDialer(os.Getenv("SMTP_HOST"), 587, os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"))

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
		return c.Redirect(http.StatusFound, "/contact")
	}

	// Victoria Pines & all of 2024
	url := "https://public.api.hospitable.com/v2/properties/" + propId + "/calendar?start_date=" + beginDate + "&end_date=" + endDate
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
	dwo := c.Request().Form["deep-work-options"]
	fmt.Println(dwo)

	massage := "no"
	meditation := "no"
	hike := "no"
	for _, d := range dwo {
		switch d {
		case "massage":
			massage = "yes"
		case "meditation":
			meditation = "yes"
		case "hike":
			hike = "yes"
		}
	}

	qs := "start=" + start + "&end=" + end + "&total=" + total + "&prop=" + prop + "&rs=" + rs + "&massage=" + massage + "&meditation=" + meditation + "&hike=" + hike
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

			d := gomail.NewDialer(os.Getenv("SMTP_HOST"), 587, os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"))

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
