package components

var insertComponent = "INSERT INTO components (id, component_type, component_path, specification_path, created_at) VALUES(?, ?, ?, ?, ?);"
var selectComponents = "SELECT * FROM components;"
var selectComponentByID = "SELECT * FROM components WHERE id=?;"
