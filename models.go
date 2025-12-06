package main

type CreateUrlRequest struct {
	URL string `json:"url"`
}

type CreateUrlResponse struct {
	ID       string `json:"id"`
	ShortURL string `json:"short_url"`
}
