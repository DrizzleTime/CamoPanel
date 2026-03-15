import { create } from "zustand";
import { apiRequest } from "../lib/api";
import type { User } from "../lib/types";

type AuthState = {
  user: User | null;
  checking: boolean;
  loadMe: () => Promise<void>;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  checking: true,
  loadMe: async () => {
    try {
      const response = await apiRequest<{ user: User }>("/api/auth/me");
      set({ user: response.user, checking: false });
    } catch {
      set({ user: null, checking: false });
    }
  },
  login: async (username, password) => {
    const response = await apiRequest<{ user: User }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    });
    set({ user: response.user });
  },
  logout: async () => {
    await apiRequest<{ ok: boolean }>("/api/auth/logout", {
      method: "POST",
    });
    set({ user: null });
  },
}));

