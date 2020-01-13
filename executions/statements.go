package executions

var insertExecutionWithNoFlowID = "INSERT INTO executions (id, build_id, component_id, created_at) VALUES(?, ?, ?, ?);"
var insertExecution = "INSERT INTO executions (id, build_id, component_id, created_at, flow_id) VALUES(?, ?, ?, ?, ?);"
