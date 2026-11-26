<div class="page-header">
    <h1>⚡ Energieverbruik scoutinggebouw</h1>
    <span class="badge">Bijgewerkt: {{.LastUpdated}}</span>
</div>

<div class="grid-2">
    <div class="card center-card gas">
        <h3>Aardgasverbruik</h3>
        <div class="big-value">{{printf "%.1f" .Energy.GasUsageTodayM3}} m³</div>
        <p class="sub-value">Vandaag tot nu toe</p>
        <p class="sub-value">Gas deze week tot nu toe: {{printf "%.1f" .Energy.GasUsageWeekM3}} m³</p>
    </div>
    <div class="card center-card grid-flow-card">
        <h3>Elektriciteitsstroom</h3>
        <div class="big-value grid-flow-value {{if gt .Energy.GridFlowWatts 0}}exporting{{else}}importing{{end}}">
            {{if gt .Energy.GridFlowWatts 0}}+{{.Energy.GridFlowWatts}} W{{else}}{{.Energy.GridFlowWatts}} W{{end}}
        </div>
        <p class="sub-value">{{if gt .Energy.GridFlowWatts 0}}Teruglevering aan het net{{else}}Afname van het net{{end}}</p>
        <p class="sub-value">Elektriciteitsverbruik deze week tot nu toe: {{printf "%.1f" .Energy.ElectricityUsageWeekKwh}} kWh</p>
    </div>
</div>

<div class="energy-chart-grid">
    <section class="card chart-card">
        <h2>Aardgasverbruik en buitentemperatuur</h2>
        <p class="chart-subtitle">{{.Energy.WeekRangeLabel}}</p>
        <div class="chart-legend">
            <span><i class="legend-swatch gas"></i>Aardgasverbruik (m³/dag)</span>
            <span><i class="legend-swatch temperature"></i>Buitentemperatuur (°C)</span>
        </div>
        <svg class="chart-svg" viewBox="0 0 960 340" role="img" aria-label="Aardgasverbruik per dag en buitentemperatuur over de afgelopen zeven dagen">
            <line class="chart-gridline" x1="72" y1="280" x2="888" y2="280"></line>
            <line class="chart-gridline" x1="72" y1="222" x2="888" y2="222"></line>
            <line class="chart-gridline" x1="72" y1="164" x2="888" y2="164"></line>
            <line class="chart-gridline" x1="72" y1="106" x2="888" y2="106"></line>
            <line class="chart-gridline" x1="72" y1="48" x2="888" y2="48"></line>
            <text class="chart-axis-label" x="10" y="170" transform="rotate(-90 10 170)">Dagverbruik (m³)</text>
            <text class="chart-axis-label" x="950" y="170" transform="rotate(90 950 170)">Temperatuur (°C)</text>
            <text class="chart-tick" x="48" y="284">0</text>
            <text class="chart-tick" x="48" y="226">1</text>
            <text class="chart-tick" x="48" y="168">2</text>
            <text class="chart-tick" x="48" y="110">3</text>
            <text class="chart-tick" x="48" y="52">4</text>
            <text class="chart-tick chart-right-tick" x="895" y="284">-5</text>
            <text class="chart-tick chart-right-tick" x="895" y="226">2,5</text>
            <text class="chart-tick chart-right-tick" x="895" y="168">10</text>
            <text class="chart-tick chart-right-tick" x="895" y="110">17,5</text>
            <text class="chart-tick chart-right-tick" x="895" y="52">25</text>
            {{range .Energy.GasDays}}
            <rect class="gas-bar" x="{{.BarX}}" y="{{.BarY}}" width="67" height="{{.BarHeight}}">
                <title>{{.DateLabel}}: {{printf "%.1f" .GasUsageM3}} m³ verbruik</title>
            </rect>
            {{end}}
            <polyline class="chart-line temperature-line" points="{{.Energy.TemperatureLine}}"></polyline>
            {{range .Energy.GasDays}}
            <text class="chart-hour date-tick" x="{{.X}}" y="304" text-anchor="middle">{{.DateLabel}}</text>
            {{end}}
        </svg>
    </section>

    <section class="card chart-card">
        <h2>Netafname, teruglevering en zoninstraling</h2>
        <p class="chart-subtitle">Netafname is positief; teruglevering is negatief · metingen per uur · vandaag</p>
        <div class="chart-legend">
            <span><i class="legend-swatch import"></i>Netafname (+ kW)</span>
            <span><i class="legend-swatch export"></i>Teruglevering (− kW)</span>
            <span><i class="legend-swatch solar-intensity"></i>Zoninstraling (W/m²)</span>
            <span><i class="legend-swatch sun-marker-legend"></i>Zonsopkomst / zonsondergang</span>
        </div>
        <svg class="chart-svg" viewBox="0 0 960 340" role="img" aria-label="Netafname boven nul, teruglevering onder nul en zoninstraling vandaag per uur">
            {{range .Energy.DaylightWindows}}
            <rect class="chart-day" x="{{.X}}" y="40" width="{{.Width}}" height="240"></rect>
            {{end}}
            <line class="chart-gridline" x1="72" y1="58" x2="888" y2="58"></line>
            <line class="chart-gridline" x1="72" y1="112" x2="888" y2="112"></line>
            <line class="chart-zero" x1="72" y1="166" x2="888" y2="166"></line>
            <line class="chart-gridline" x1="72" y1="220" x2="888" y2="220"></line>
            <line class="chart-gridline" x1="72" y1="274" x2="888" y2="274"></line>
            <text class="chart-axis-label" x="10" y="170" transform="rotate(-90 10 170)">Netvermogen (kW)</text>
            <text class="chart-axis-label" x="950" y="170" transform="rotate(90 950 170)">Zoninstraling (W/m²)</text>
            <text class="chart-tick" x="40" y="62">+6</text>
            <text class="chart-tick" x="40" y="116">+3</text>
            <text class="chart-tick" x="48" y="170">0</text>
            <text class="chart-tick" x="40" y="224">−3</text>
            <text class="chart-tick" x="40" y="278">−6</text>
            <text class="chart-tick chart-right-tick" x="895" y="278">0</text>
            <text class="chart-tick chart-right-tick" x="895" y="170">500</text>
            <text class="chart-tick chart-right-tick" x="895" y="62">1000</text>
            {{range .Energy.GridPoints}}
            <rect class="import-bar" x="{{.BarX}}" y="{{.ImportY}}" width="14" height="{{.ImportHeight}}">
                <title>{{.HourLabel}}:00 netafname</title>
            </rect>
            <rect class="export-bar" x="{{.BarX}}" y="166" width="14" height="{{.ExportHeight}}">
                <title>{{.HourLabel}}:00 teruglevering</title>
            </rect>
                {{if .HourLabel}}<text class="chart-hour" x="{{.X}}" y="304" text-anchor="middle">{{.HourLabel}}</text>{{end}}
            {{end}}
            <polyline class="chart-line solar-line" points="{{.Energy.SolarLine}}"></polyline>
            {{range .Energy.SunriseMarkers}}
            <line class="sun-marker" x1="{{.}}" y1="40" x2="{{.}}" y2="280"></line>
            {{end}}
            {{range .Energy.SunsetMarkers}}
            <line class="sun-marker" x1="{{.}}" y1="40" x2="{{.}}" y2="280"></line>
            {{end}}
        </svg>
    </section>
</div>
