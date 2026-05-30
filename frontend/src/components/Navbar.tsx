import React, { useState } from "react";
import { Sparkles, Library as LibIcon, Piano, LogIn } from "lucide-react";

interface NavbarProps {
  activeTab: "generate" | "library";
  setActiveTab: (tab: "generate" | "library") => void;
}

export default function Navbar({ activeTab, setActiveTab }: NavbarProps) {
  const [walletConnected, setWalletConnected] = useState(false);
  const [walletAddress, setWalletAddress] = useState("");
  const [isConnecting, setIsConnecting] = useState(false);

  const handleConnectWallet = () => {
    if (walletConnected) {
      setWalletConnected(false);
      setWalletAddress("");
      return;
    }

    setIsConnecting(true);
    setTimeout(() => {
      const mockHex = Array.from({ length: 6 }, () =>
        Math.floor(Math.random() * 16).toString(16)
      ).join("");
      setWalletAddress(`0x${mockHex.slice(0, 4)}...${mockHex.slice(-4).toUpperCase()}`);
      setWalletConnected(true);
      setIsConnecting(false);
    }, 1200);
  };

  return (
    <nav className="w-full bg-surface/60 backdrop-blur-xl border-b border-white/10 shadow-sm" id="top-nav">
      <div className="flex justify-between items-center px-lg py-md w-full">
        {/* Brand Logo */}
        <div 
          onClick={() => setActiveTab("generate")}
          className="text-headline-lg font-display font-extrabold bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent cursor-pointer select-none active:scale-95 transition-transform"
        >
          MidiMind
        </div>

        {/* Middle Navigation Routes */}
        <div className="hidden md:flex items-center gap-xl">
          <button
            onClick={() => setActiveTab("generate")}
            className={`flex items-center gap-2 pb-1 text-sm font-medium tracking-wide border-b-2 hover:text-white transition-all cursor-pointer ${
              activeTab === "generate"
                ? "border-primary text-primary"
                : "border-transparent text-on-surface-variant hover:border-white/20"
            }`}
          >
            <Sparkles className="w-4 h-4" />
            Generate
          </button>
          <button
            onClick={() => setActiveTab("library")}
            className={`flex items-center gap-2 pb-1 text-sm font-medium tracking-wide border-b-2 hover:text-white transition-all cursor-pointer ${
              activeTab === "library"
                ? "border-primary text-primary"
                : "border-transparent text-on-surface-variant hover:border-white/20"
            }`}
          >
            <LibIcon className="w-4 h-4" />
            Library
          </button>
        </div>

        {/* Right Controls */}
        <div className="flex items-center gap-md">
          
        </div>
      </div>
    </nav>
  );
}
