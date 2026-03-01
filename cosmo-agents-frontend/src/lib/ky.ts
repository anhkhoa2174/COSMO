import ky, { HTTPError } from 'ky';
import { statusCodeErrorMap } from '@/network/errors/httpErrors';
import { authService } from '@/services/authService';

export async function authMiddleware(request: Request): Promise<Request> {
  const accessToken = await authService.getAccessToken();

  if (accessToken && !request.headers.get('Authorization')) {
    request.headers.set('Authorization', `Bearer ${accessToken}`);
  }

  return request;
}

export async function handleAuthError(
  error: HTTPError
): Promise<Response | void> {
  try {
    // Attempt to refresh the token
    await authService.refreshToken();
    const accessToken = await authService.getAccessToken();

    if (accessToken) {
      const originalRequest = error.request.clone();
      originalRequest.headers.set('Authorization', `Bearer ${accessToken}`);
      return fetch(originalRequest);
    }
  } catch (refreshError) {
    // If refresh fails, logout and redirect to login
    await authService.logout();
    throw refreshError;
  }
}

const PUBLIC_URL = process.env.NEXT_PUBLIC_SERVER_BASE_URL;
const SERVER_URL = process.env.SERVER_BASE_URL || PUBLIC_URL;

export const kyClient = ky.create({
  prefixUrl: SERVER_URL,
  headers: {
    Accept: 'application/json',
  },
  hooks: {
    beforeRequest: [authMiddleware],
    beforeError: [
      async (error) => {
        const body = await error.response.json<any>();
        const errorMessage =
          body.error?.message ||
          body.detail[0]?.msg ||
          body.detail ||
          'Unknown error occurred';

        // Map HTTP status code to error class
        if (error.response.status in statusCodeErrorMap) {
          throw new statusCodeErrorMap[error.response.status](errorMessage);
        }

        // Generic error
        throw new Error(
          `Something went wrong: ${error.response.status} : ${errorMessage}`
        );
      },
    ],
    afterResponse: [
      async (request, options, response) => {
        if (response.status === 401) {
          return handleAuthError(new HTTPError(response, request, options));
        }
        if (response.status === 403) {
          await authService.logout();
        }
        return response;
      },
    ],
  },
  retry: {
    limit: 2,
    methods: ['get', 'put', 'head', 'delete', 'options', 'trace'],
    statusCodes: [408, 413, 429, 500, 502, 503, 504],
  },
});
