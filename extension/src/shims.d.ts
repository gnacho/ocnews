// The SDK ships the host Tailwind classes as a plain css file; the bundler
// resolves it, but the type checker needs the module declared.
declare module '*.css'
