package controllers

import (
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

type DashboardController struct {
	beego.Controller
}

func (c *DashboardController) Get() {
	loc, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		loc = time.FixedZone("CET", 3600)
	}
	c.Data["CurrentTime"] = time.Now().In(loc).Format("15:04:05")
	c.Data["ActiveSlide"] = "energy"
	c.TplName = "index.tpl"
}
