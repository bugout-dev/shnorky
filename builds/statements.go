package builds

var insertBuild = "INSERT INTO builds (id, component_id, created_at) VALUES(?, ?, ?);"
var selectBuilds = "SELECT * FROM builds;"
var selectBuildByID = "SELECT * FROM builds WHERE id=?;"
var selectBuildsByComponentID = "SELECT * FROM builds WHERE component_id=?;"
var deleteBuildByID = "DELETE FROM builds WHERE id=?;"
var deleteBuildsByComponentID = "DELETE FROM builds WHERE component_id=?"
