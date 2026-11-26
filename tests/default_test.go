package test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/beego/beego/v2/core/logs"

	_ "LSSKiosk/routers"

	beego "github.com/beego/beego/v2/server/web"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	_, file, _, _ := runtime.Caller(0)
	apppath, _ := filepath.Abs(filepath.Dir(filepath.Join(file, ".."+string(filepath.Separator))))
	beego.TestBeegoInit(apppath)
}

// TestBeego is a sample to run an endpoint test
func TestBeego(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	logs.Trace("testing", "TestBeego", "Code[%d]\n%s", w.Code, w.Body.String())

	Convey("Subject: Test Station Endpoint\n", t, func() {
		Convey("Status Code Should Be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The Result Should Not Be Empty", func() {
			So(w.Body.Len(), ShouldBeGreaterThan, 0)
		})
	})
}

func TestDashboardRoutes(t *testing.T) {
	routes := []struct {
		path          string
		expectedItems []string
	}{
		{path: "/", expectedItems: []string{`lang="nl"`, "Scoutingdashboard", "id=\"slide-energy\"", "id=\"slide-planes\"", "id=\"slide-weather\"", "Dashboardpagina's"}},
		{path: "/energy", expectedItems: []string{"Aardgasverbruik", "Vandaag tot nu toe", "Gas deze week tot nu toe:", "Elektriciteitsstroom", "Elektriciteitsverbruik deze week tot nu toe:", " kWh", "Aardgasverbruik en buitentemperatuur", "Aardgasverbruik (m³/dag)", "Netafname, teruglevering en zoninstraling", "Netafname is positief; teruglevering is negatief", "Uurmetingen ·", "netafname"}},
		{path: "/weather", expectedItems: []string{"Gedeeltelijk bewolkt", "Komende uren", "Gevoelstemperatuur", "Neerslagkans"}},
		{path: "/static/img/aircraft-placeholder.svg", expectedItems: []string{"Vliegtuigafbeelding niet beschikbaar"}},
	}

	for _, route := range routes {
		t.Run(route.path, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, route.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			beego.BeeApp.Handlers.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("GET %s returned status %d", route.path, response.Code)
			}

			body := response.Body.String()
			for _, content := range route.expectedItems {
				if !strings.Contains(body, content) {
					t.Errorf("GET %s response is missing expected content %q", route.path, content)
				}
			}
			if route.path == "/energy" {
				if count := strings.Count(body, `<rect class="gas-bar"`); count != 7 {
					t.Errorf("GET /energy rendered %d gas bars, want one for each of the last seven dates", count)
				}
				if !strings.Contains(body, `class="chart-line solar-line"`) {
					t.Error("GET /energy is missing the solar intensity line")
				}
				if count := strings.Count(body, `<rect class="import-bar"`); count != 25 {
					t.Errorf("GET /energy rendered %d import bars, want 25 hourly samples", count)
				}
				if count := strings.Count(body, `class="sun-marker"`); count != 2 {
					t.Errorf("GET /energy rendered %d sunrise/sunset markers, want 2 on the solar chart", count)
				}
				if count := strings.Count(body, `class="chart-hour date-tick"`); count != 7 {
					t.Errorf("GET /energy rendered %d date labels, want 7 for the gas chart", count)
				}
				if strings.Contains(body, "Net Grid Balance") {
					t.Error("energy page still contains the separate net grid balance card")
				}
				if strings.Contains(body, "Current Building Usage") {
					t.Error("grid flow widget still reports building energy usage")
				}
				if !strings.Contains(body, `class="big-value grid-flow-value`) {
					t.Error("current grid flow value is missing the large headline style")
				}
				if !(strings.Contains(body, "+300 W") && strings.Contains(body, "Teruglevering aan het net")) &&
					!(strings.Contains(body, "-300 W") && strings.Contains(body, "Afname van het net")) {
					t.Error("current grid flow is missing a matching signed export/import value and label")
				}
				gasChartStart := strings.Index(body, "Aardgasverbruik en buitentemperatuur")
				gridChartStart := strings.Index(body, "Netafname, teruglevering en zoninstraling")
				if gasChartStart >= 0 && gridChartStart > gasChartStart {
					gasChart := body[gasChartStart:gridChartStart]
					if strings.Contains(gasChart, "sun-marker") || strings.Contains(gasChart, "zonsopkomst") || strings.Contains(gasChart, "zonsondergang") {
						t.Error("natural gas chart still contains sunrise or sunset markers")
					}
				}
			}
		})
	}
}
