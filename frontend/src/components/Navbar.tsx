import React, { useState } from "react";
import { Sparkles, Library as LibIcon, Piano, Wallet, LogIn, Disc } from "lucide-react";

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
          <button className="hidden sm:flex items-center gap-1.5 text-on-surface-variant hover:text-primary transition-colors cursor-pointer text-sm font-medium px-md py-sm">
            <LogIn className="w-4 h-4" />
            Login
          </button>
          
          <button
            onClick={handleConnectWallet}
            disabled={isConnecting}
            className={`relative flex items-center gap-2 bg-gradient-to-r ${
              walletConnected 
                ? "from-emerald-500/20 to-teal-500/20 border border-emerald-500/50 text-emerald-400"
                : "from-primary-container to-secondary-container text-on-primary-container"
            } px-lg py-sm rounded-full font-bold hover:scale-105 active:scale-95 transition-all shadow-lg neon-glow cursor-pointer`}
          >
            {isConnecting ? (
              <>
                <Disc className="w-4 h-4 animate-spin text-on-primary-container" />
                <span>Syncing...</span>
              </>
            ) : walletConnected ? (
              <>
                <div className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse" />
                <span className="font-mono text-xs">{walletAddress}</span>
              </>
            ) : (
              <>
                <Wallet className="w-4 h-4" />
                <span className="text-xs">Connect Wallet</span>
              </>
            )}
          </button>
        </div>
      </div>
    </nav>
  );
}
