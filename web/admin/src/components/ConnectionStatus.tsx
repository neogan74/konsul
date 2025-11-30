import { Wifi, WifiOff } from 'lucide-react';
import { useWebSocket } from '../contexts/WebSocketContext';

export default function ConnectionStatus() {
  const { isConnected } = useWebSocket();

  return (
    <div className="flex items-center gap-2">
      {isConnected ? (
        <>
          <Wifi size={16} className="text-green-400" />
          <span className="text-xs text-green-400 hidden md:inline">Live</span>
        </>
      ) : (
        <>
          <WifiOff size={16} className="text-red-400" />
          <span className="text-xs text-red-400 hidden md:inline">Disconnected</span>
        </>
      )}
    </div>
  );
}