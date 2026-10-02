package rules

var pyEntries = []CatalogEntry{
	{Name: "request.args", Category: CategorySource, Language: "python"},
	{Name: "request.form", Category: CategorySource, Language: "python"},
	{Name: "request.GET", Category: CategorySource, Language: "python"},
	{Name: "request.POST", Category: CategorySource, Language: "python"},
	{Name: "cursor.execute", Category: CategorySink, SinkKind: SinkSQL, Language: "python"},
	{Name: "execute", Category: CategorySink, SinkKind: SinkSQL, Language: "python"},
	{Name: "os.system", Category: CategorySink, SinkKind: SinkCommand, Language: "python"},
	{Name: "subprocess.Popen", Category: CategorySink, SinkKind: SinkCommand, Language: "python"},
	{Name: "subprocess.run", Category: CategorySink, SinkKind: SinkCommand, Language: "python"},
	{Name: "open", Category: CategorySink, SinkKind: SinkFile, Language: "python"},
	{Name: "escape", Category: CategorySanitizer, Language: "python"},
	{Name: "sanitize", Category: CategorySanitizer, Language: "python"},
	{Name: "pathlib.Path.resolve", Category: CategorySanitizer, Language: "python"},
	{Name: "os.path.abspath", Category: CategorySanitizer, Language: "python"},
}
