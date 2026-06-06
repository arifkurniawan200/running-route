package service

import (
	"fmt"

	"github.com/arifkurniawan200/running-route/internal/client"
	"github.com/arifkurniawan200/running-route/internal/model"
)

type Geocoding struct {
	client *client.Nominatim
}

func NewGeocoding(client *client.Nominatim) *Geocoding {
	return &Geocoding{client: client}
}

func (s *Geocoding) Search(query string) (*model.NominatimResponse, error) {
	return s.client.Search(query)
}

func (s *Geocoding) Reverse(lat, lng float64) (string, error) {
	resp, err := s.client.ReverseGeocode(lat, lng)
	if err != nil {
		return "", err
	}
	if len(*resp) > 0 {
		return (*resp)[0].DisplayName, nil
	}
	return fmt.Sprintf("%.4f, %.4f", lat, lng), nil
}
