export function readFragmentParams(): URLSearchParams {
  return new URLSearchParams(window.location.hash.slice(1))
}

// A one-time token (invite, password reset) shouldn't stay in the address
// bar or browser history once it's been read: clearing the fragment right
// after reading it keeps it out of history entries synced elsewhere.
export function clearURLFragment(): void {
  window.history.replaceState(null, '', window.location.pathname + window.location.search)
}
