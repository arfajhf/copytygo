package cli

import "path/filepath"

// generatedFileIntro adds a short orientation note without annotating every line.
// Resource and job discovery markers are kept in their original locations.
func generatedFileIntro(path string) string {
	switch filepath.Base(filepath.Dir(path)) {
	case "controllers":
		return "// ==========================================\n" +
			"// Application Controller\n" +
			"// Customize request handling and validation here.\n" +
			"// ==========================================\n\n"
	case "services":
		return "// ==========================================\n" +
			"// Application Service\n" +
			"// Add reusable business logic here.\n" +
			"// ==========================================\n\n"
	case "models":
		return "// Model fields define the data used by your application.\n\n"
	case "middleware":
		return "// Middleware runs before the request handler.\n\n"
	case "jobs":
		return "// A job handles background work dispatched to the queue.\n\n"
	case "listeners":
		return "// Event listeners handle events in the application.\n\n"
	case "seeders":
		return "// Seeders populate initial application data.\n\n"
	case "factories":
		return "// Factories create sample data for development and tests.\n\n"
	case "mails":
		return "// Mail messages define recipients and email content.\n\n"
	default:
		return ""
	}
}
