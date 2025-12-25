package domain

import "time"

// URLClick represents a single click/access event for analytics
// This is a separate entity from URL because it represents a different concept
// One URL can have many URLClicks (one-to-many relationship)
type URLClick struct {
	ID          int64     // Auto-incrementing ID
	URLID       string    // Foreign key to URL
	ClickedAt   time.Time // When the click occurred
	IPAddress   string    // IP address of the visitor
	UserAgent   string    // Browser/client information
	Referer     string    // Where the visitor came from
	CountryCode string    // Geolocation: country (e.g., "US")
	City        string    // Geolocation: city
}

// NewURLClick creates a new click event
func NewURLClick(urlID, ipAddress, userAgent, referer string) *URLClick {
	return &URLClick{
		URLID:     urlID,
		ClickedAt: time.Now(),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Referer:   referer,
	}
}

// WithGeolocation adds geolocation data to the click event
func (c *URLClick) WithGeolocation(countryCode, city string) *URLClick {
	c.CountryCode = countryCode
	c.City = city
	return c
}
