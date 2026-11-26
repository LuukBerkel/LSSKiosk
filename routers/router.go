package routers

import (
	"LSSKiosk/controllers"
	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	beego.Router("/", &controllers.DashboardController{})
	beego.Router("/energy", &controllers.EnergyController{})
	beego.Router("/planes", &controllers.PlanesController{})
	beego.Router("/weather", &controllers.WeatherController{})
}
