package controllers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestFetchNearbyAircraftFiltersSortsAndConverts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"ac":[
			{"hex":"far001","flight":"FAR   ","desc":"Cessna 172","alt_baro":5000,"gs":100,"lat":51.61,"lon":4.483695864613846},
			{"hex":"near01","flight":"NEAR   ","desc":"Piper P210","alt_baro":10000,"gs":150,"track":205.89,"baro_rate":736,"squawk":"0176","category":"A3","lat":51.52,"lon":4.483695864613846},
			{"hex":"ground","flight":"GROUND  ","t":"A21N","alt_baro":"ground","gs":0,"lat":51.51106866490034,"lon":4.483695864613846},
			{"hex":"ground0","flight":"GROUND0 ","alt_baro":0,"gs":0,"lat":51.51106866490034,"lon":4.483695864613846},
			{"hex":"lowair","flight":"LOWAIR  ","alt_baro":0,"gs":80,"lat":51.51106866490034,"lon":4.483695864613846},
			{"hex":"outside","flight":"OUTSIDE","lat":52.0,"lon":4.483695864613846},
			{"hex":"nopos","flight":"NOPOS"}
		]}`)
	}))
	defer server.Close()

	planes, err := fetchNearbyAircraft(
		context.Background(),
		server.Client(),
		server.URL,
		aircraftLatitude,
		aircraftLongitude,
		aircraftRadiusKm,
	)
	if err != nil {
		t.Fatalf("fetchNearbyAircraft returned an error: %v", err)
	}
	if len(planes) != 3 {
		t.Fatalf("fetchNearbyAircraft returned %d planes, want 3 airborne planes within range", len(planes))
	}
	if planes[0].Callsign != "LOWAIR" || planes[0].Distance != 0 {
		t.Errorf("closest plane = %+v, want LOWAIR at the center; grounded aircraft should be omitted", planes[0])
	}
	if planes[1].Callsign != "NEAR" || planes[1].Altitude != 3048 || planes[1].Speed != 278 {
		t.Errorf("converted NEAR plane = %+v, want 3048 m and 278 km/u", planes[1])
	}
	if !planes[1].HasHeading || planes[1].Heading != 206 ||
		!planes[1].HasVertical || planes[1].VerticalMove != "Stijgt" || planes[1].VerticalRate != 224 {
		t.Errorf("flight details = %+v, want heading 206 and climbing at 224 m/min", planes[1])
	}
	if planes[1].Squawk != "0176" || planes[1].Category != "A3" || planes[1].Position != "51.52000, 4.48370" {
		t.Errorf("aircraft identification = %+v, want squawk, category, and rounded position", planes[1])
	}
	if planes[2].Callsign != "FAR" {
		t.Errorf("third plane = %+v, want FAR after distance sorting", planes[2])
	}
}

func TestFetchNearbyAircraftRejectsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := fetchNearbyAircraft(
		context.Background(),
		server.Client(),
		server.URL,
		aircraftLatitude,
		aircraftLongitude,
		aircraftRadiusKm,
	)
	if err == nil {
		t.Fatal("fetchNearbyAircraft did not report an unsuccessful API status")
	}
}

func TestFetchAircraftPhotoReturnsLargeThumbnailAndCredit(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if r.URL.Path != "/photos/hex/4852ed" {
			t.Errorf("photo request path = %q, want /photos/hex/4852ed", r.URL.Path)
		}
		if r.UserAgent() != "LSSKiosk/1.0 (+https://github.com/LuukBerkel/LSSKiosk)" {
			t.Errorf("photo request user agent = %q, want a contactable app user agent", r.UserAgent())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"photos":[{"thumbnail":{"src":"https://t.plnspttrs.net/small.jpg"},"thumbnail_large":{"src":"https://t.plnspttrs.net/large.jpg"},"link":"https://www.planespotters.net/photo/123","photographer":"A. Photographer"}]}`)
	}))
	defer server.Close()

	photo, err := fetchAircraftPhoto(context.Background(), server.Client(), server.URL+"/photos/hex", "4852ed")
	if err != nil {
		t.Fatalf("fetchAircraftPhoto returned an error: %v", err)
	}
	if photo.ImageURL != "https://t.plnspttrs.net/large.jpg" {
		t.Errorf("photo URL = %q, want large thumbnail URL", photo.ImageURL)
	}
	if photo.PageURL != "https://www.planespotters.net/photo/123" || photo.Photographer != "A. Photographer" {
		t.Errorf("photo attribution = %+v, want photo link and photographer", photo)
	}

	if _, err := fetchAircraftPhoto(context.Background(), server.Client(), server.URL+"/photos/hex", "4852ed"); err != nil {
		t.Fatalf("cached fetchAircraftPhoto returned an error: %v", err)
	}
	if count := requestCount.Load(); count != 1 {
		t.Errorf("photo service was called %d times, want one call due to image caching", count)
	}
}

func TestResolveAircraftPhotoFallsBackToRegistrationAndAirportData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/photos/hex/486869", "/photos/reg/PH-GCD":
			_, _ = fmt.Fprint(w, `{"photos":[]}`)
		case "/photos/reg/PH-TEST":
			_, _ = fmt.Fprint(w, `{"photos":[{"thumbnail":{"src":"https://t.plnspttrs.net/reg.jpg"},"link":"https://www.planespotters.net/photo/456","photographer":"Reg Spotter"}]}`)
		case "/ac_thumb.json":
			if got := r.URL.Query().Get("m"); got != "486869" {
				t.Errorf("airport-data m query = %q, want 486869", got)
			}
			_, _ = fmt.Fprint(w, `{"status":200,"count":1,"data":[{"image":"http://airport-data.com/images/aircraft/thumbnails/000/123/123456.jpg","link":"http://airport-data.com/aircraft/photo/000123456","photographer":"AD Spotter"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	regPhoto, err := resolveAircraftPhoto(
		context.Background(),
		server.Client(),
		server.URL+"/photos/hex",
		server.URL+"/photos/reg",
		server.URL+"/ac_thumb.json",
		"486869",
		"PH-TEST",
	)
	if err != nil {
		t.Fatalf("resolveAircraftPhoto (reg fallback) returned error: %v", err)
	}
	if regPhoto.ImageURL != "https://t.plnspttrs.net/reg.jpg" || regPhoto.Source != "Planespotters.net" {
		t.Errorf("reg fallback photo = %+v, want Planespotters reg thumbnail", regPhoto)
	}

	adPhoto, err := resolveAircraftPhoto(
		context.Background(),
		server.Client(),
		server.URL+"/photos/hex",
		server.URL+"/photos/reg",
		server.URL+"/ac_thumb.json",
		"486869",
		"PH-GCD",
	)
	if err != nil {
		t.Fatalf("resolveAircraftPhoto (airport-data fallback) returned error: %v", err)
	}
	if adPhoto.ImageURL != "https://airport-data.com/images/aircraft/thumbnails/000/123/123456.jpg" ||
		adPhoto.PageURL != "https://airport-data.com/aircraft/photo/000123456" ||
		adPhoto.Source != "Airport-Data.com" ||
		adPhoto.Photographer != "AD Spotter" {
		t.Errorf("airport-data fallback photo = %+v, want upgraded HTTPS Airport-Data photo", adPhoto)
	}
}
