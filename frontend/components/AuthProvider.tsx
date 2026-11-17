"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  apiFetch,
  fetchCurrentUser,
  fetchMyProfile,
  type MeProfile,
  type User,
} from "@/lib/api/client";

export type AuthState = {
  user: User | null;
  profile: MeProfile | null;
  ready: boolean;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
};

export const AuthContext = createContext<AuthState | null>(null);

export function useAuth(): AuthState {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return value;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [profile, setProfile] = useState<MeProfile | null>(null);
  const [ready, setReady] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const next = await fetchCurrentUser();
      setUser(next);
      if (!next) {
        setProfile(null);
        return;
      }
      try {
        setProfile(await fetchMyProfile());
      } catch {
        setProfile(null);
      }
    } catch {
      setUser(null);
      setProfile(null);
    }
  }, []);

  const logout = useCallback(async () => {
    await apiFetch("/v1/auth/logout", { method: "POST" });
    setUser(null);
    setProfile(null);
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const next = await fetchCurrentUser();
        if (cancelled) {
          return;
        }
        setUser(next);
        if (!next) {
          setProfile(null);
          return;
        }
        try {
          const me = await fetchMyProfile();
          if (!cancelled) {
            setProfile(me);
          }
        } catch {
          if (!cancelled) {
            setProfile(null);
          }
        }
      } catch {
        if (!cancelled) {
          setUser(null);
          setProfile(null);
        }
      } finally {
        if (!cancelled) {
          setReady(true);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const value = useMemo(
    () => ({ user, profile, ready, refresh, logout }),
    [user, profile, ready, refresh, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
