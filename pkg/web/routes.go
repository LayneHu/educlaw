package web

// This file documents the route structure.
// Route registration is handled in server.go setupRoutes().
//
// Routes:
//   GET  /                          -> redirect to /student
//   GET  /student                   -> student chat page
//   GET  /parent                    -> parent dashboard page
//   GET  /teacher                   -> teacher portal page
//   POST /api/chat                  -> send a chat message
//   GET  /api/chat/stream/:session  -> SSE stream for a session
//   GET  /api/student/:id/summary   -> student knowledge summary
//   GET  /api/parent/:id/report     -> parent progress report
//   GET  /api/teacher/:id/class-report -> class analytics
//   POST /api/onboard               -> register new actor
//   GET  /api/actors/:type          -> list actors by type
