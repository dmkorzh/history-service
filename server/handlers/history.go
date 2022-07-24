package handlers

import (
	"net/http"

	"github.com/dmkorzh/history-service/server/model"
	"github.com/gin-gonic/gin"
)

// History godoc
// @Summary      Call history
// @Description  History calls and counts
// @Produce      json
// @Param        department  path      string  true   "Department ID"
// @Param        start       query     string  true   "Start of period (format:20211005T063228Z)"
// @Param        end         query     string  true   "End of period (format:20211005T063228Z)"
// @Param        limit       query     int     false  "Calls response limit (default - 500)"
// @Success      200         {object}  model.HistoryResult
// @Failure      400         {object}  HTTPError
// @Failure      500         {object}  HTTPError
// @Router       /history/{department} [get]
func (ctrl *Controller) History(c *gin.Context) {
	params := model.HistoryParams{
		Department: c.Param("department"),
		Limit:      500,
		Comment:    c.Request.URL.String(),
	}
	if err := c.ShouldBindQuery(&params); err != nil {
		httpError(c, http.StatusBadRequest, err)
		return
	}

	endpoint := endpoint(c)
	calls, counts, err := model.GetHistory(&params, ctrl.Database.Driver, ctrl.Templates[endpoint])
	if err != nil {
		httpError(c, http.StatusInternalServerError, err)
		return
	}
	response := &model.HistoryResult{
		Counts: counts,
		Calls:  calls,
	}
	c.JSON(http.StatusOK, response)
}
