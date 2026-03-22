package report

type CategorizedFindings struct {
	Blocking []Finding
	Critical []Finding
	Warning  []Finding
	Info     []Finding
}

func Categorize(findings []Finding, minSeverity Severity) CategorizedFindings {
	var cf CategorizedFindings
	for _, f := range findings {
		sev, err := ParseSeverity(f.Severity)
		if err != nil || !sev.MeetsMinimum(minSeverity) {
			continue
		}
		switch sev {
		case SeverityBlocking:
			cf.Blocking = append(cf.Blocking, f)
		case SeverityCritical:
			cf.Critical = append(cf.Critical, f)
		case SeverityWarning:
			cf.Warning = append(cf.Warning, f)
		case SeverityInfo:
			cf.Info = append(cf.Info, f)
		}
	}
	return cf
}

func (cf CategorizedFindings) Total() int {
	return len(cf.Blocking) + len(cf.Critical) + len(cf.Warning) + len(cf.Info)
}

func (cf CategorizedFindings) HasBlockersOrCritical() bool {
	return len(cf.Blocking) > 0 || len(cf.Critical) > 0
}
