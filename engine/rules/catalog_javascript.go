package rules

var jsEntries = []CatalogEntry{
	{Name: "req.query", Category: CategorySource, Language: "javascript"},
	{Name: "req.body", Category: CategorySource, Language: "javascript"},
	{Name: "req.params", Category: CategorySource, Language: "javascript"},
	{Name: "db.query", Category: CategorySink, SinkKind: SinkSQL, Language: "javascript"},
	{Name: "connection.query", Category: CategorySink, SinkKind: SinkSQL, Language: "javascript"},
	{Name: "client.query", Category: CategorySink, SinkKind: SinkSQL, Language: "javascript"},
	{Name: "child_process.exec", Category: CategorySink, SinkKind: SinkCommand, Language: "javascript"},
	{Name: "exec", Category: CategorySink, SinkKind: SinkCommand, Language: "javascript"},
	{Name: "execSync", Category: CategorySink, SinkKind: SinkCommand, Language: "javascript"},
	{Name: "fs.readFile", Category: CategorySink, SinkKind: SinkFile, Language: "javascript"},
	{Name: "fs.writeFile", Category: CategorySink, SinkKind: SinkFile, Language: "javascript"},
	{Name: "escape", Category: CategorySanitizer, Language: "javascript"},
	{Name: "sanitize", Category: CategorySanitizer, Language: "javascript"},
	{Name: "path.resolve", Category: CategorySanitizer, Language: "javascript"},
}
