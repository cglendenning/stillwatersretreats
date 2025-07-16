# Stillwaters Retreats

## Synopsis
Stillwaters Retreats offers deep thinking retreats at Big Bear Lake, CA, designed for creatives, builders, and deep thinkers who crave long stretches of uninterrupted time to focus, create, and recharge. The retreats are hosted in modern, serene mountain cabins, providing an environment free from distractions, noise, and the pressures of daily life. The site and its offerings are crafted by Craig, a tech professional and AirBnB superhost with over 300 5-star reviews, who understands the importance of flow, clarity, and intentional space for meaningful work.

> "Our retreats are designed for deep thinkers who crave long stretches of uninterrupted time to ponder and engage in solitary creative pursuits. Nestled in beautiful, natural surroundings, our cabins provide a serene environment free from the pressures of micromanagement, high-pressure sales, and corporate team-building exercises." ([howitworks.html](templates/howitworks.html))

## Key Features
- **Cabin Retreats**: Two modern, 5-star reviewed cabins (Bear View Cove and Victoria Pines) in Big Bear Lake, CA, available for solo or team retreats.
- **Flexible Structure**: Choose from a simple cabin stay or add-ons like on-demand video courses, live virtual coaching, or onsite coaching with Craig.
- **Deep Work Add-ons**: Optional on-site massage, guided meditation, and nature hikes to enhance your retreat experience.
- **Unscheduled Flow**: No rigid schedules—guests are encouraged to follow their own creative rhythms, with optional guidance available.
- **Support for Builders**: The retreats are especially suited for creatives, coders, writers, entrepreneurs, and anyone needing a sanctuary for deep work.
- **Blog**: A collection of essays and posts addressing the challenges of focus, creativity, and productivity in a noisy world, and how the right environment can unlock breakthroughs.

## Site Components

### Frontend
- **Static HTML/CSS**: The main site is rendered with HTML templates and styled with CSS for a clean, modern look.
- **Vue.js App**: The `/still-waters/` directory contains a Vue 3 app (with Vite, Pinia, and Vue Router) for interactive or experimental frontend features.
- **Responsive Design**: The site is mobile-friendly and visually appealing across devices.

### Backend
- **Go (Golang) Server**: The backend is built with Go using the Echo web framework, serving static assets, HTML templates, and handling API endpoints.
- **Booking & Calendar**: Integrates with the Hospitable API to fetch property and calendar data for real-time availability.
- **Checkout & Payments**: Uses Stripe for secure online payments and checkout sessions.
- **Contact & Booking Forms**: Handles form submissions with Google reCAPTCHA Enterprise for spam protection and sends emails via SMTP (Amazon SES or similar).
- **Blog System**: Blog posts are stored as JSON files and rendered via templates.

### Integrations
- **Stripe**: For secure payment processing and checkout flows.
- **Hospitable API**: For property management, calendar, and booking data.
- **Google reCAPTCHA Enterprise**: For spam and bot protection on forms.
- **Amazon SES (or SMTP)**: For sending transactional emails (contact and booking confirmations).
- **Firebase Analytics**: For site analytics and event tracking.
- **Google Analytics (gtag.js)**: For additional analytics and conversion tracking.
- **YouTube**: Embedded videos for additional content and guidance.

## Booking Flow
1. **Browse Cabins**: View details and photos of available cabins.
2. **Select Dates & Add-ons**: Choose your stay dates and any retreat structure or deep work add-ons.
3. **Review & Payment**: Review your booking and pay securely via Stripe.
4. **Confirmation**: Receive email confirmation and details for your retreat.

## Project Structure
- `main.go`: Go backend server, routes, and API integrations.
- `templates/`: HTML templates for all main pages (home, cabins, how it works, pricing, blog, booking, etc.).
- `assets/`: Images and videos for the site and blog.
- `blog/`: Blog post content in JSON format.
- `static/`: Static files (CSS, JS, images).
- `still-waters/`: Vue 3 frontend app (optional/experimental).

## Development & Deployment
- **Backend**: Go 1.22+, Echo framework.
- **Frontend**: Static HTML/CSS, Vue 3 (Vite) for app components.
- **Deployment**: Automated via GitHub Actions to a live Ubuntu server, with Nginx for serving static and dynamic content.

## Philosophy
Stillwaters Retreats is built on the belief that environment is everything for deep work and creative breakthroughs. The site, cabins, and retreat structure are all designed to help you reclaim your focus, find clarity, and do your best work—free from the chaos of everyday life.

---

For more information, visit [stillwatersretreats.com](https://stillwatersretreats.com) or contact Craig via the site contact form. 
