import {useState, useEffect, useMemo} from 'react';
import "./App.css";
import {
    ListDevices,
    StartMonitoring,
    StopMonitoring,
    GetStatusSnapshot,
    AddDevice,
    DeleteDevice,
    ImportDevices 
} from "../wailsjs/go/main/App"

function formatDateTime(dt){
    if(!dt) return "-";
    const d = new Date(dt);
    if (Number.isNaN(d.getTime())) return String(dt);
    return d.toLocaleString()
}

function normalizeStatus(st){
    if(!st) return "";
    return String(st).toUpperCase();
}

function App(){
    const [devices, setDevices] = useState([]);
    const [statusMap, setStatusMap] = useState({})
    const [polling, setPolling] = useState(false)

    // Form State
    const [name, setName] = useState("Device-1");
    const [ip, setIp] = useState("192.168.70.99");
    const [interval, setIntervalSec] = useState(10);
    const [mode, setMode] = useState("pingcmd");
    const [tcpPort, setTcpPort] = useState(80);

    const refreshDevices = async ()=>{
        try {
            const devs = await ListDevices();
            setDevices(devs || []);
            
        } catch (err) {
            console.error("Error in ListDevices", err)
        }
    }

    useEffect(()=>{
        refreshDevices()
    },[])


    //Load device once on Start
    useEffect(()=>{
        (
            async () =>{
                try{
                    const devs = await ListDevices()
                    setDevices(devs || []);
                }catch(e){
                    console.error("List Devices failed", e)
                }
            }
        )()
    })
    // Poll snapshot periodically
    useEffect(()=>{
        if (!polling) return;

        let alive = true;
        const tick = async ()=>{
            try {
                const snap = await GetStatusSnapshot();
                console.log(snap)
                if(alive) setStatusMap(snap);
            } catch (err) {
               console.error("Get Snapshot failed", err);
            }
        }
        tick();
        const iv = setInterval(tick, 1000);
        return()=>{
            alive = false;
            clearInterval(iv)
        };
    },[polling]);

    const rows = useMemo(()=>{
        return (devices|| []).map((d)=>{
            const st = statusMap?.[d.id];
            const lastStatus = st?.LastStatus || st?.lastStatus || "";
            const lastSeen =  st?.LastSeen || st?.lastSeen || "";
            const lastRTT = st?.LastRTT || st?.lastRTT || 0;
            const lastReason = st?.LastReason || st?.lastReason || "";
            return {
                ...d,
                lastStatus,
                lastSeen,
                lastRTT,
                lastReason
            }
        })
    }, [devices, statusMap]);

   

    const onStart = async ()=>{
        try {
            await StartMonitoring();
            setPolling(true);
        } catch (err) {
           console.error("Failed to start Monitoring", err) 
        }
    };


    const onStop = async ()=>{
        try {
            await StopMonitoring()
            setPolling(false)
        } catch (err) {
            console.error("Failed to stop monitoring", err)
        }
    };

    const onAdd = async ()=>{
        try {
            const payLoad = {
                Name: name.trim(),
                IP: ip.trim(),
                Interval: Number(interval) || 10,
                Mode: mode,
                TcpPort: mode==='tcp'?tcpPort || 80:0,
                Enabled:true
            }
            
            await AddDevice(payLoad);
            await refreshDevices();
        } catch (err) {
            console.error("AddDevice Failed", err)
            alert(err?.message|| err)
        }
    }

    const onDelete = async (id)=>{
        try {
            await DeleteDevice(id);
            await refreshDevices();
        } catch (err) {
            console.error("DeleteDevice Failed", err);
            alert(err?.message || err);
        }
    }


    return(
        <div style={{ padding: 16, fontFamily: "Segoe UI, sans-serif" }}>
            <h2 style={{ marginTop: 0 }}>ELV Monitor (V1)</h2>
            <div style={{ display: "flex", gap: 8, marginBottom: 12 }}>
            <button onClick={onStart}>Start</button>
            <button onClick={onStop}>Stop</button>
            <button onClick={refreshDevices}> Refresh Devices</button>
            <div style={{ marginLeft: 12, opacity: 0.7 }}>
                Devices : {devices.length} | Polling: {polling? "ON":"OFF"}
            </div>
            </div>
            {/* Add Device*/}
            
            <div
                style={{
                    border: "1px solid #ddd",
                    borderRadius: 8,
                    padding: 12,
                    marginBottom: 12,
                    display: "flex",
                    gap: 8,
                    flexWrap: "wrap",
                    alignItems: "end",
                }}
            >
                <div style={{display: "flex", flexDirection: "column", gap: 4}}>
                    <label>Name</label>
                    <input type="text" value={name} onChange={(e)=>{ setName(e.target.value)}} />
                </div>
                <div style={{display: "flex", flexDirection: "column", gap: 4}}>
                    <label>IP</label>
                    <input type="text" value={ip} onChange={(e)=>{ setIp(e.target.value)}} />
                </div>
                <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                    <label htmlFor="">Interval (sec)</label>
                    <input 
                        type='Number'
                        min = '1'
                        value={interval}
                        onChange={(e)=>{setIntervalSec(e.target.value)}}
                        style={{ width: 120 }}
                    >
                    </input>
                </div>
                <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                    <label>Mode</label>
                    <select value={mode} onChange={(e)=>{ setMode(e.target.value)}}>
                        <option value="pingcmd">PingCmd (default)</option>
                        <option value="tcp">TCP</option>
                        <option value="icmp">ICMP (later)</option>
                        <option value="auto">Auto</option>
                    </select>
                </div>
                {
                    mode === "tcp" && (
                        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                            <label htmlFor="">TCP Port</label>
                            <input 
                            type="Number"
                            min="1"
                            max="65535" 
                            value={tcpPort} 
                            onChange={(e)=>{setTcpPort(e.target.value)}} />
                        </div>
                    )
                }
                <button onClick={onAdd}>Add Device</button>
                <div style={{ marginBottom: 12 }}>
                    <label style={{ marginRight: 8 }}>Import CSV:</label>
                    <input 
                        type="file"
                        accept=".csv,text/csv"
                        onChange = {async(e)=>{
                            const file  = e.target.files?.[0]
                            if (!file) return;
                            const text = await file.text();
                            const devs = parseDevicesCsv(text);
                            const result = await ImportDevices(devs);
                            const total = result?.total ?? result?.Total ?? 0;
                            const added = result?.added ?? result?.Added ?? 0;
                            const updated = result?.updated ?? result?.Updated ?? 0;
                            const failed = result?.failed ?? result?.Failed ?? 0;
                            const errors = result?.errors ?? result?.Errors ?? [];
                            alert( `Imported: total=${total}, added=${added}, updated=${updated}, failed=${failed}\n` +(errors?.length ? errors.slice(0, 8).join("\n") : ""));
                            await refreshDevices();
                        }}
                    />
                </div>
            </div>


             <div style={{ overflowX: "auto" }}>
                <table
                    style={{
                        width: "100%",
                        borderCollapse: "collapse",
                        minWidth: 900,
                    }}
                >
                    <thead>
                        <tr>
                            <th style={th}>Name</th>
                            <th style={th}>IP</th>
                            <th style={th}>Interval</th>
                            <th style={th}>Mode</th>
                            <th style={th}>Status</th>
                            <th style={th}>Last Seen</th>
                            <th style={th}>RTT</th>
                            <th style={th}>Reason</th>
                            <th style={th}>Action</th>
                        </tr>
                    </thead>
                    <tbody>

                        {
                           
                            rows.map((r)=>{
                                const st = normalizeStatus(r.lastStatus);
                                const up = st === "UP";
                                const down = st === "DOWN";
                                return(
                                    <tr key={r.Id}>
                                        <td style={td}>{r.name}</td>
                                        <td style={td}>{r.ip}</td>
                                        <td style={td}>{r.interval}s</td>
                                        <td style={td}>{r.mode}</td>
                                        <td style={td}>
                                            <span
                                            style={{
                                                padding: "2px 10px",
                                                borderRadius: 999,
                                                color: "#fff",
                                                background: up ? "green" : down ? "red" : "gray",
                                                display: "inline-block",
                                                minWidth: 70,
                                                textAlign: "center",
                                            }}>
                                                {st || "-"}
                                            </span>
                                        </td>
                                        <td style={td}>{formatDateTime(r.lastSeen)}</td>
                                        <td style={td}>{formatRTT(r.lastRTT)}</td>
                                        <td style={td}>{r.lastReason || "-"}</td>
                                        <td style={td}>
                                            <button onClick={()=>{onDelete(r.id)}}>Delete</button>
                                        </td>
                                    </tr>
                                );
                            })
                        }
                        {rows.length === 0 && (
                        <tr>
                            <td style={td} colSpan={9}>
                            No devices yet. Add One
                            </td>
                        </tr>
                        )}
                    </tbody>
                </table>
             </div>
              <p style={{ marginTop: 12, opacity: 0.7 }}>
                V1: state-change logging only (UP↔DOWN). Default mode: PingCmd.
            </p>
         </div>
    )

}
const th = {
  textAlign: "left",
  borderBottom: "1px solid #ccc",
  padding: "8px 10px",
  fontWeight: 600,
};

const td = {
  borderBottom: "1px solid #eee",
  padding: "8px 10px",
};

function parseDevicesCsv(text){
    const lines = text.split(/\r?\n/).map((l) => l.trim()).filter((l) => l.length > 0);
    if (lines.length < 2) return [];
    const header = parseCsvLine(lines[0]).map((h)=> h.toLowerCase())
    const idx = (name)=>header.indexOf(name.toLowerCase())
    const iName = idx("name");
    const iIP = idx("ip");
    const iInterval = idx("interval");
    const iMode = idx("mode");
    const iTcpPort = idx("tcpPort");
    const iEnabled = idx("enabled");
    const devs = [];
    for (let r = 1; r < lines.length; r++) {
        const cols = parseCsvLine(lines[r]);
        const name = iName >= 0 ? cols[iName] : "";
        const ip = iIP >= 0 ? cols[iIP] : "";
        if (!name || !ip) continue;
        const interval = iInterval >= 0 ? Number(cols[iInterval]) : 10;
        const mode = iMode >= 0 ? (cols[iMode] || "pingcmd") : "pingcmd";
        const tcpPort = iTcpPort >= 0 ? Number(cols[iTcpPort]) : 0;
        const enabledRaw = iEnabled >= 0 ? (cols[iEnabled] || "true") : "true";
        const enabled = ["true", "1", "yes", "y"].includes(enabledRaw.toLowerCase());
        devs.push({
            name, ip,
            interval: Number.isFinite(interval) && interval > 0 ? interval : 10,
            mode,
            tcpPort: mode === "tcp" ? (Number.isFinite(tcpPort) && tcpPort > 0 ? tcpPort : 80) : 0,
            enabled,
         })
    }

    return devs;
}

function parseCsvLine(line) {
  // Handles commas + quotes (basic)
  const out = [];
  let cur = "";
  let inQuotes = false;

  for (let i = 0; i < line.length; i++) {
    const ch = line[i];

    if (ch === '"') {
      if (inQuotes && line[i + 1] === '"') {
        cur += '"';
        i++;
      } else {
        inQuotes = !inQuotes;
      }
      continue;
    }

    if (ch === "," && !inQuotes) {
      out.push(cur.trim());
      cur = "";
      continue;
    }

    cur += ch;
  }
  out.push(cur.trim());
  return out;
}

function formatRTT(val){
    if (!val) return "-";
    if (typeof val === "string") return val;
    const ns = Number(val);
    if (!Number.isFinite(ns)) return "-";
    const sec = ns / 1e9;
    if (sec < 1) {
        const ms = ns / 1e6;
        return `${ms.toFixed(ms < 10 ? 2 : ms < 100 ? 1 : 0)} ms`;
    }

    return `${sec.toFixed(sec < 10 ? 2 : 1)} s`;
}

export default App
