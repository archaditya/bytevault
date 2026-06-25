import { create } from "zustand";
import { User } from "@/types";
import { currentUser } from "@/lib/mock";

interface AuthState {
  user: User;
  isAuthenticated: boolean;
  setUser: (user: User) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: currentUser,
  isAuthenticated: true,
  setUser: (user) => set({ user }),
}));
