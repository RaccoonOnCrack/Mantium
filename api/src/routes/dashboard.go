package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/dashboard"
)

// DashboardRoutes sets the routes for the dashboard.
func DashboardRoutes(group *gin.RouterGroup) {
	{
		group.GET("/dashboard/configs", GetDashboardConfigs)
		group.GET("/dashboard/last_update", GetLastUpdate)
		group.PATCH("/dashboard/configs/columns", UpdateDashboardColumns)
		group.GET("/dashboard/last_background_error", GetLastBackgroundError)
		group.DELETE("/dashboard/last_background_error", DeleteLastBackgroundError)
	}
}

// @Summary Get the dashboard configs
// @Description Returns the dashboard configs read from the configs.json file.
// @Success 200 {object} dashboard.Configs
// @Produce json
// @Router /dashboard/configs [get]
func GetDashboardConfigs(c *gin.Context) {
	configs, err := dashboard.GetConfigsFromFile(config.GlobalConfigs.ConfigsFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error while loading configs file: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"configs": configs.Dashboard})
}

// @Summary Get the last update date
// @Description Returns the last time a resource that should trigger an update in the iframe/dashboard was updated. Usually used to update the dashboard when an event not triggered by the user occurs.
// @Success 200 {object} responseMessage
// @Produce json
// @Router /dashboard/last_update [get]
func GetLastUpdate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": dashboard.GetLastUpdateDashboard(),
	})
}

// @Summary Update dashboard columns
// @Description Update the dashboard columns in the configs.json file.
// @Success 200 {object} responseMessage
// @Produce json
// @Param columns query int true "New number of columns." Example(5)
// @Router /dashboard/configs/columns [patch]
func UpdateDashboardColumns(c *gin.Context) {
	columnsStr := c.Query("columns")
	if columnsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing 'columns' parameter"})
		return
	}
	columns, err := strconv.Atoi(columnsStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to convert 'columns' parameter into number"})
		return
	}

	configsFilePath := config.GlobalConfigs.ConfigsFilePath

	configs, err := dashboard.GetConfigsFromFile(configsFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error while loading configs file: %s", err.Error())})
		return
	}

	configs.Dashboard.Columns = columns

	updatedConfigs, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error while updating configs file: %s", err.Error())})
		return
	}

	err = os.WriteFile(configsFilePath, updatedConfigs, 0o644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error while updating configs file: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configs updated successfully"})
}

// @Summary Get the last background error
// @Description Returns the last error that happened in the background. Usually used to display the error in the dashboard.
// @Success 200 {object} dashboard.BackgroundError
// @Produce json
// @Router /dashboard/last_background_error [get]
func GetLastBackgroundError(c *gin.Context) {
	lastError := dashboard.GetLastBackgroundError()
	c.JSON(http.StatusOK, gin.H{"message": lastError.Message, "time": lastError.Time.Format("2006-01-02 15:04:05")})
}

// @Summary Delete the last background error
// @Description Deletes the last error that happened in the background. Usually used to clear the error in the dashboard.
// @Success 200 {object} responseMessage
// @Produce json
// @Router /dashboard/last_background_error [delete]
func DeleteLastBackgroundError(c *gin.Context) {
	dashboard.DeleteLastBackgroundError()
	c.JSON(http.StatusOK, gin.H{"message": "Last background error deleted"})
}
