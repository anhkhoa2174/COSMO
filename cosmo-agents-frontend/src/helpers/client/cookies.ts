import Cookies from 'js-cookie';

export const getToken = () => {
  const agent_access_token = Cookies.get('agent_access_token');
  const access_token = Cookies.get('access_token');
  const session_name = Cookies.get('session_name');
  return { agent_access_token, access_token, session_name };
};

export const setToken = ({
  agent_access_token,
  session_name,
}: {
  agent_access_token?: string;
  session_name?: string;
}) => {
  if (agent_access_token) {
    Cookies.set('agent_access_token', agent_access_token, {
      ...(process.env.NODE_ENV === 'development'
        ? { httpOnly: false, secure: false }
        : { httpOnly: true, secure: true }),
      expires: 1, // 1 day
      path: '/',
      sameSite: 'lax',
    });
  }
  if (session_name) {
    Cookies.set('session_name', session_name, {
      ...(process.env.NODE_ENV === 'development'
        ? { httpOnly: false, secure: false }
        : { httpOnly: true, secure: true }),
      expires: 1, // 1 day
      path: '/',
      sameSite: 'lax',
    });
  }
};
