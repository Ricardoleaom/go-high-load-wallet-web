// Where the console finds the services it talks to. Vite fixes these at
// build time; the defaults are the backend's and the load lab's local
// addresses.
export const config = {
  apiUrl: import.meta.env.VITE_API_URL ?? 'http://localhost:8080',
  loadlabUrl: import.meta.env.VITE_LOADLAB_URL ?? 'http://localhost:8090',
  oidcAuthority: import.meta.env.VITE_OIDC_AUTHORITY ?? 'http://localhost:8081/realms/wagering',
  oidcClientId: import.meta.env.VITE_OIDC_CLIENT_ID ?? 'wallet-console',
} as const
