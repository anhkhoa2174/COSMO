import { cookies } from 'next/headers';
import { generateId } from '..';

export const getToken = async () => {
  const cookieStore = cookies();
  const agent_access_token = cookieStore.get('agent_access_token');
  const access_token = cookieStore.get('access_token');
  const session_name = cookieStore.get('session_name');
  return {
    agent_access_token: agent_access_token?.value,
    access_token: access_token?.value,
    session_name: session_name?.value,
  };
};

export const setToken = async ({
  agent_access_token,
  session_name,
}: {
  agent_access_token?: string;
  session_name?: string;
}) => {
  const cookieStore = cookies();
  if (agent_access_token) {
    cookieStore.set('agent_access_token', agent_access_token, {
      ...(process.env.NODE_ENV === 'development'
        ? { httpOnly: false, secure: false }
        : { httpOnly: true, secure: true }),
      expires: 1 / 96,
      path: '/',
      sameSite: 'lax',
    });
  }
  if (session_name) {
    cookieStore.set('session_name', session_name, {
      ...(process.env.NODE_ENV === 'development'
        ? { httpOnly: false, secure: false }
        : { httpOnly: true, secure: true }),
      expires: 1 / 96,
      path: '/',
      sameSite: 'lax',
    });
  }
};

/**
 * Returns a usable Coze token, minting one when the cookie is missing.
 *
 * Pass `forceRefresh` after Coze has rejected the cached token: the cookie
 * outlives the token it holds, so without this the caller stays stuck on a
 * dead token until the cookie itself expires.
 */
export const getAccessToken = async (forceRefresh = false) => {
  const {
    access_token,
    agent_access_token: cachedAgentToken,
    session_name,
  } = await getToken();
  const agent_access_token = forceRefresh ? undefined : cachedAgentToken;

  let session_nameRes = session_name;

  if (!session_name) {
    session_nameRes = generateId();
    setToken({ session_name: session_nameRes });
  }

  if (!agent_access_token) {
    try {
      const res = await fetch(
        `${process.env.SERVER_BASE_URL || process.env.NEXT_PUBLIC_SERVER_BASE_URL}/v1/auth/coze/token`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${access_token}`,
          },
          body: JSON.stringify({
            session_name: session_nameRes,
            duration_seconds: 3600,
          }),
        }
      );

      if (!res.ok) {
        throw new Error('Failed to get access token.');
      }

      const resJson = await res.json();
      const agent_access_tokenRes = resJson?.data?.access_token;
      setToken({
        agent_access_token: agent_access_tokenRes,
        session_name: session_nameRes,
      });

      return {
        agent_access_token: agent_access_tokenRes,
        access_token,
        session_name: session_nameRes,
      };
    } catch (err) {
      throw new Error('Error getting access token.');
    }
  }
  return {
    agent_access_token,
    access_token,
    session_name,
  };
};
