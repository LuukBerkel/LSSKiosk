package controllers

import (
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

type DashboardController struct {
	beego.Controller
}

func (c *DashboardController) Get() {
	c.Data["LastUpdated"] = time.Now().Format("15:04")
	c.Data["ActiveSlide"] = "energy"
	c.TplName = "index.tpl"
}
