package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
)

const (
	aircraftLatitude    = 51.51106866490034
	aircraftLongitude   = 4.483695864613846
	aircraftRadiusKm    = 25.0
	aircraftAPIURL      = "https://opendata.adsb.fi/api/v3/lat/51.51106866490034/lon/4.483695864613846/dist/25"
	aircraftPhotoAPI    = "https://api.planespotters.net/pub/photos/hex"
	aircraftPhotoRegAPI = "https://api.planespotters.net/pub/photos/reg"
	airportDataPhotoAPI = "https://airport-data.com/api/ac_thumb.json"
)

var (
	aircraftHTTPClient      = &http.Client{Timeout: 10 * time.Second}
	aircraftPhotoHTTPClient = &http.Client{Timeout: 4 * time.Second}
	aircraftPhotoCache      sync.Map
)

type PlanesController struct {
	beego.Controller
}

type planeData struct {
	Callsign     string
	Hex          string
	AircraftType string
	Registration string
	ImageURL     string
	ImagePageURL string
	ImageSource  string
	Photographer string
	Altitude     int
	Speed        int
	Distance     float64
	Heading      int
	HasHeading   bool
	VerticalRate int
	VerticalMove string
	HasVertical  bool
	Squawk       string
	Category     string
	Position     string
}

type adsbResponse struct {
	Aircraft *[]adsbAircraft `json:"ac"`
}

type adsbAircraft struct {
	Hex          string          `json:"hex"`
	Flight       string          `json:"flight"`
	Registration string          `json:"r"`
	Type         string          `json:"t"`
	Description  string          `json:"desc"`
	AltitudeBaro json.RawMessage `json:"alt_baro"`
	GroundSpeed  *float64        `json:"gs"`
	Track        *float64        `json:"track"`
	BaroRate     *float64        `json:"baro_rate"`
	Squawk       string          `json:"squawk"`
	Category     string          `json:"category"`
	Latitude     *float64        `json:"lat"`
	Longitude    *float64        `json:"lon"`
}

func (c *PlanesController) Get() {
	ctx := c.Ctx.Request.Context()
	planes, err := fetchNearbyAircraft(
		ctx,
		aircraftHTTPClient,
		aircraftAPIURL,
		aircraftLatitude,
		aircraftLongitude,
		aircraftRadiusKm,
	)
	if err != nil {
		logs.Error("Unable to fetch nearby aircraft from ADS-B.fi: %v", err)
		c.Data["PlanesError"] = "Live vliegtuiggegevens zijn tijdelijk niet beschikbaar."
	}

	if err == nil && len(planes) > 0 && (planes[0].Hex != "" || planes[0].Registration != "") {
		photo, photoErr := resolveAircraftPhoto(
			ctx,
			aircraftPhotoHTTPClient,
			aircraftPhotoAPI,
			aircraftPhotoRegAPI,
			airportDataPhotoAPI,
			planes[0].Hex,
			planes[0].Registration,
		)
		if photoErr != nil {
			logs.Warn("Unable to fetch aircraft photo for %s: %v", planes[0].Hex, photoErr)
		} else {
			planes[0].ImageURL = photo.ImageURL
			planes[0].ImagePageURL = photo.PageURL
			planes[0].ImageSource = photo.Source
			planes[0].Photographer = photo.Photographer
		}
	}

	c.Data["LastUpdated"] = time.Now().Format("15:04:05")
	c.Data["Planes"] = planes
	if len(planes) > 0 {
		c.Data["ClosestPlane"] = &planes[0]
	}
	c.TplName = "planes.tpl"
}

func fetchNearbyAircraft(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	centerLatitude, centerLongitude, radiusKm float64,
) ([]planeData, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create ADS-B.fi request: %w", err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request ADS-B.fi aircraft: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("ADS-B.fi returned HTTP %d", response.StatusCode)
	}

	var data adsbResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode ADS-B.fi response: %w", err)
	}
	if data.Aircraft == nil {
		return nil, fmt.Errorf("ADS-B.fi response is missing aircraft data")
	}

	planes := make([]planeData, 0, len(*data.Aircraft))
	for _, aircraft := range *data.Aircraft {
		if isOnGround(aircraft.AltitudeBaro, aircraft.GroundSpeed) {
			continue
		}
		if aircraft.Latitude == nil || aircraft.Longitude == nil {
			continue
		}
		if *aircraft.Latitude < -90 || *aircraft.Latitude > 90 ||
			*aircraft.Longitude < -180 || *aircraft.Longitude > 180 {
			continue
		}

		distance := distanceBetweenKm(
			centerLatitude,
			centerLongitude,
			*aircraft.Latitude,
			*aircraft.Longitude,
		)
		if distance > radiusKm {
			continue
		}

		callsign := strings.TrimSpace(aircraft.Flight)
		if callsign == "" {
			callsign = strings.ToUpper(aircraft.Hex)
		}
		aircraftType := strings.TrimSpace(aircraft.Description)
		if aircraftType == "" {
			aircraftType = strings.TrimSpace(aircraft.Type)
		}
		if aircraftType == "" {
			aircraftType = "Onbekend vliegtuigtype"
		}

		speedKmh := 0
		if aircraft.GroundSpeed != nil {
			speedKmh = int(math.Round(*aircraft.GroundSpeed * 1.852))
		}
		plane := planeData{
			Callsign:     callsign,
			Hex:          strings.TrimSpace(aircraft.Hex),
			AircraftType: aircraftType,
			Registration: strings.TrimSpace(aircraft.Registration),
			Altitude:     altitudeInMeters(aircraft.AltitudeBaro),
			Speed:        speedKmh,
			Distance:     math.Round(distance*10) / 10,
			Squawk:       strings.TrimSpace(aircraft.Squawk),
			Category:     strings.TrimSpace(aircraft.Category),
			Position:     fmt.Sprintf("%.5f, %.5f", *aircraft.Latitude, *aircraft.Longitude),
		}
		if aircraft.Track != nil {
			plane.Heading = int(math.Round(*aircraft.Track)) % 360
			plane.HasHeading = true
		}
		if aircraft.BaroRate != nil {
			verticalRate := int(math.Round(*aircraft.BaroRate * 0.3048))
			plane.VerticalRate = int(math.Abs(float64(verticalRate)))
			switch {
			case verticalRate > 0:
				plane.VerticalMove = "Stijgt"
			case verticalRate < 0:
				plane.VerticalMove = "Daalt"
			default:
				plane.VerticalMove = "Horizontaal"
			}
			plane.HasVertical = true
		}
		planes = append(planes, plane)
	}

	sort.Slice(planes, func(i, j int) bool {
		if planes[i].Distance == planes[j].Distance {
			return planes[i].Callsign < planes[j].Callsign
		}
		return planes[i].Distance < planes[j].Distance
	})
	return planes, nil
}

func isOnGround(altitude json.RawMessage, groundSpeed *float64) bool {
	var altitudeStatus string
	if json.Unmarshal(altitude, &altitudeStatus) == nil {
		altitudeStatus = strings.TrimSpace(altitudeStatus)
		if strings.EqualFold(altitudeStatus, "ground") {
			return true
		}
		feet, err := strconv.ParseFloat(altitudeStatus, 64)
		return err == nil && feet <= 0 && groundSpeed != nil && *groundSpeed <= 0
	}

	var feet float64
	return json.Unmarshal(altitude, &feet) == nil &&
		feet <= 0 && groundSpeed != nil && *groundSpeed <= 0
}

type aircraftPhoto struct {
	ImageURL     string
	PageURL      string
	Source       string
	Photographer string
}

type cachedAircraftPhoto struct {
	Photo     aircraftPhoto
	ExpiresAt time.Time
}

type planespottersResponse struct {
	Photos []struct {
		Thumbnail      photoThumbnail `json:"thumbnail"`
		ThumbnailLarge photoThumbnail `json:"thumbnail_large"`
		PageURL        string         `json:"link"`
		Photographer   string         `json:"photographer"`
	} `json:"photos"`
}

type photoThumbnail struct {
	URL string `json:"src"`
}

type airportDataResponse struct {
	Status int `json:"status"`
	Count  int `json:"count"`
	Data   []struct {
		Image        string `json:"image"`
		PageURL      string `json:"link"`
		Photographer string `json:"photographer"`
	} `json:"data"`
}

// resolveAircraftPhoto fetches an aircraft photo using Planespotters (hex, then registration)
// and falls back to Airport-Data.com when Planespotters has no matching image.
func resolveAircraftPhoto(
	ctx context.Context,
	client *http.Client,
	hexEndpoint, regEndpoint, airportDataEndpoint, hex, registration string,
) (aircraftPhoto, error) {
	hex = strings.TrimSpace(hex)
	registration = strings.TrimSpace(registration)
	cacheKey := "resolve|" + hexEndpoint + "|" + strings.ToLower(hex) + "|" + strings.ToUpper(registration)
	if cached, ok := aircraftPhotoCache.Load(cacheKey); ok {
		result := cached.(cachedAircraftPhoto)
		if time.Now().Before(result.ExpiresAt) {
			return result.Photo, nil
		}
		aircraftPhotoCache.Delete(cacheKey)
	}

	var lastErr error
	if hex != "" && !strings.HasPrefix(hex, "~") {
		photo, err := fetchAircraftPhoto(ctx, client, hexEndpoint, hex)
		if err != nil {
			lastErr = err
		} else if photo.ImageURL != "" {
			aircraftPhotoCache.Store(cacheKey, cachedAircraftPhoto{Photo: photo, ExpiresAt: time.Now().Add(time.Hour)})
			return photo, nil
		}
	}

	if registration != "" {
		photo, err := fetchAircraftPhoto(ctx, client, regEndpoint, registration)
		if err != nil {
			lastErr = err
		} else if photo.ImageURL != "" {
			aircraftPhotoCache.Store(cacheKey, cachedAircraftPhoto{Photo: photo, ExpiresAt: time.Now().Add(time.Hour)})
			return photo, nil
		}
	}

	if airportDataEndpoint != "" && ((hex != "" && !strings.HasPrefix(hex, "~")) || registration != "") {
		photo, err := fetchAirportDataPhoto(ctx, client, airportDataEndpoint, hex, registration)
		if err != nil {
			lastErr = err
		} else if photo.ImageURL != "" {
			aircraftPhotoCache.Store(cacheKey, cachedAircraftPhoto{Photo: photo, ExpiresAt: time.Now().Add(time.Hour)})
			return photo, nil
		}
	}

	if lastErr != nil {
		return aircraftPhoto{}, lastErr
	}
	aircraftPhotoCache.Store(cacheKey, cachedAircraftPhoto{ExpiresAt: time.Now().Add(time.Hour)})
	return aircraftPhoto{}, nil
}

func fetchAircraftPhoto(ctx context.Context, client *http.Client, endpoint, identifier string) (aircraftPhoto, error) {
	requestURL := strings.TrimRight(endpoint, "/") + "/" + url.PathEscape(identifier)
	if cached, ok := aircraftPhotoCache.Load(requestURL); ok {
		result := cached.(cachedAircraftPhoto)
		if time.Now().Before(result.ExpiresAt) {
			return result.Photo, nil
		}
		aircraftPhotoCache.Delete(requestURL)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return aircraftPhoto{}, fmt.Errorf("create aircraft photo request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "LSSKiosk/1.0 (+https://github.com/LuukBerkel/LSSKiosk)")

	response, err := client.Do(request)
	if err != nil {
		return aircraftPhoto{}, fmt.Errorf("request aircraft photo: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return aircraftPhoto{}, fmt.Errorf("aircraft photo service returned HTTP %d", response.StatusCode)
	}

	var data planespottersResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&data); err != nil {
		return aircraftPhoto{}, fmt.Errorf("decode aircraft photo response: %w", err)
	}
	if len(data.Photos) == 0 {
		aircraftPhotoCache.Store(requestURL, cachedAircraftPhoto{ExpiresAt: time.Now().Add(time.Hour)})
		return aircraftPhoto{}, nil
	}

	photo := data.Photos[0]
	imageURL := photo.ThumbnailLarge.URL
	if !isHTTPSURLForDomain(imageURL, "plnspttrs.net") {
		imageURL = photo.Thumbnail.URL
	}
	if !isHTTPSURLForDomain(imageURL, "plnspttrs.net") {
		return aircraftPhoto{}, fmt.Errorf("aircraft photo response has no trusted thumbnail URL")
	}

	pageURL := ""
	if isHTTPSURLForDomain(photo.PageURL, "planespotters.net") {
		pageURL = photo.PageURL
	}
	result := aircraftPhoto{
		ImageURL:     imageURL,
		PageURL:      pageURL,
		Source:       "Planespotters.net",
		Photographer: strings.TrimSpace(photo.Photographer),
	}
	aircraftPhotoCache.Store(requestURL, cachedAircraftPhoto{
		Photo:     result,
		ExpiresAt: time.Now().Add(time.Hour),
	})
	return result, nil
}

func fetchAirportDataPhoto(ctx context.Context, client *http.Client, endpoint, hex, registration string) (aircraftPhoto, error) {
	query := url.Values{}
	if hex != "" && !strings.HasPrefix(hex, "~") {
		query.Set("m", hex)
	} else if registration != "" {
		query.Set("r", registration)
	}
	query.Set("n", "1")
	requestURL := endpoint + "?" + query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return aircraftPhoto{}, fmt.Errorf("create airport-data photo request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "LSSKiosk/1.0 (+https://github.com/LuukBerkel/LSSKiosk)")

	response, err := client.Do(request)
	if err != nil {
		return aircraftPhoto{}, fmt.Errorf("request airport-data photo: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return aircraftPhoto{}, fmt.Errorf("airport-data photo service returned HTTP %d", response.StatusCode)
	}

	var data airportDataResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&data); err != nil {
		return aircraftPhoto{}, fmt.Errorf("decode airport-data photo response: %w", err)
	}
	if len(data.Data) == 0 {
		return aircraftPhoto{}, nil
	}

	photo := data.Data[0]
	imageURL := upgradeDomainHTTPToHTTPS(strings.TrimSpace(photo.Image), "airport-data.com")
	if !isHTTPSURLForDomain(imageURL, "airport-data.com") {
		return aircraftPhoto{}, fmt.Errorf("airport-data photo response has no trusted image URL")
	}
	pageURL := upgradeDomainHTTPToHTTPS(strings.TrimSpace(photo.PageURL), "airport-data.com")
	if !isHTTPSURLForDomain(pageURL, "airport-data.com") {
		pageURL = ""
	}
	return aircraftPhoto{
		ImageURL:     imageURL,
		PageURL:      pageURL,
		Source:       "Airport-Data.com",
		Photographer: strings.TrimSpace(photo.Photographer),
	}, nil
}

func upgradeDomainHTTPToHTTPS(rawURL, domain string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return rawURL
	}
	if parsed.Scheme == "http" && (parsed.Host == domain || strings.HasSuffix(parsed.Host, "."+domain)) {
		parsed.Scheme = "https"
		return parsed.String()
	}
	return rawURL
}

func isHTTPSURLForDomain(rawURL, domain string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return false
	}
	return parsed.Host == domain || strings.HasSuffix(parsed.Host, "."+domain)
}

func altitudeInMeters(altitude json.RawMessage) int {
	var feet float64
	if err := json.Unmarshal(altitude, &feet); err != nil {
		var value string
		if json.Unmarshal(altitude, &value) != nil {
			return 0
		}
		feet, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return 0
		}
	}
	return int(math.Round(feet * 0.3048))
}

func distanceBetweenKm(latitude1, longitude1, latitude2, longitude2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRadians := math.Pi / 180
	latitudeDelta := (latitude2 - latitude1) * toRadians
	longitudeDelta := (longitude2 - longitude1) * toRadians
	latitude1 *= toRadians
	latitude2 *= toRadians

	a := math.Sin(latitudeDelta/2)*math.Sin(latitudeDelta/2) +
		math.Cos(latitude1)*math.Cos(latitude2)*
			math.Sin(longitudeDelta/2)*math.Sin(longitudeDelta/2)
	return 2 * earthRadiusKm * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
