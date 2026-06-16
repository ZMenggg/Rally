const I18N = {zh: {
    subtitle:'多 VPS 带宽聚合代理', stopped:'已停止', running:'运行中',
    nodesLabel:'节点', online:'在线',
    tabDashboard:'仪表盘', tabNodes:'节点管理', tabLogs:'日志', tabConfig:'配置',
    totalNodes:'总节点', activeLabel:'在线',downSpeed:'下行速率',upSpeed:'上行速率',totalDown:'总下行',totalUp:'总上行', disabledLabel:'已禁用',
    nodeStatus:'节点状态',
    nameCol:'名称', typeCol:'类型', serverCol:'服务器',
    statusCol:'状态', activeConnsCol:'活跃', enabledCol:'启用',
    portCol:'端口', passwordCol:'密码', actionsCol:'操作',
    addNode:'+ 添加节点', refresh:'刷新',
    saveConfigBtn:'保存配置', reloadBtn:'重载',
    addNodeTitle:'添加节点', editNodeTitle:'编辑节点',
    nameLabel:'名称', typeLabel:'类型', serverLabel:'服务器',
    portLabel:'端口', passwordLabel:'密码', sniLabel:'SNI',
    cipherLabel:'加密方式', uuidLabel:'UUID', flowLabel:'流控',
    downMbpsLabel:'下行带宽 (Mbps)', upMbpsLabel:'上行带宽 (Mbps)',
    addNodeBtn:'添加节点', editNodeBtn:'保存', cancelBtn:'取消',
    nodeUpdated:'节点已更新', nodeAdded:'节点已添加',
    nodeDeleted:'节点已删除', configSaved:'配置已保存',
    configReloaded:'配置已重载',
    deleteConfirm:'确定要删除此节点吗？',
    onlineStatus:'在线', offlineStatus:'离线',
    failedLoadDashboard:'加载失败: ',
    failedLoadNodes:'加载节点失败: ',
    failedLoadConfig:'加载配置失败: ',
    edit:'编辑', delete:'删除', todo:'需重启 rally run 生效',
    clearBtn:'清空', autoScroll:'自动滚动',
    toggleOn:'开启', toggleOff:'关闭', nodeToggled:'节点开关已切换',
  },
  en: {
    subtitle:'Multi-VPS Bandwidth Aggregation', stopped:'Stopped', running:'Running',
    nodesLabel:'Nodes', online:'online',
    tabDashboard:'Dashboard', tabNodes:'Nodes', tabLogs:'Logs', tabConfig:'Config',
    totalNodes:'Total Nodes', activeLabel:'Active',downSpeed:'Down',upSpeed:'Up',totalDown:'Total Down',totalUp:'Total Up', disabledLabel:'Disabled',
    nodeStatus:'Node Status',
    nameCol:'Name', typeCol:'Type', serverCol:'Server',
    statusCol:'Status', activeConnsCol:'Active', enabledCol:'Enabled',
    portCol:'Port', passwordCol:'Password', actionsCol:'Actions',
    addNode:'+ Add Node', refresh:'Refresh',
    saveConfigBtn:'Save Config', reloadBtn:'Reload',
    addNodeTitle:'Add Node', editNodeTitle:'Edit Node',
    nameLabel:'Name', typeLabel:'Type', serverLabel:'Server',
    portLabel:'Port', passwordLabel:'Password', sniLabel:'SNI',
    cipherLabel:'Cipher', uuidLabel:'UUID', flowLabel:'Flow',
    downMbpsLabel:'Down Mbps', upMbpsLabel:'Up Mbps',
    addNodeBtn:'Add Node', editNodeBtn:'Save', cancelBtn:'Cancel',
    nodeUpdated:'Node updated', nodeAdded:'Node added',
    nodeDeleted:'Node deleted', configSaved:'Config saved',
    configReloaded:'Config reloaded',
    deleteConfirm:'Delete this node?',
    onlineStatus:'Online', offlineStatus:'Offline',
    failedLoadDashboard:'Failed to load: ',
    failedLoadNodes:'Failed to load nodes: ',
    failedLoadConfig:'Failed to load config: ',
    edit:'Edit', delete:'Delete', todo:'Restart rally run',
    clearBtn:'Clear', autoScroll:'Auto Scroll',
    toggleOn:'On', toggleOff:'Off', nodeToggled:'Node toggled',
  },
};

let currentLang = localStorage.getItem('rally_lang') || 'zh';

function t(k){return I18N[currentLang]?.[k]||I18N.en[k]||k}

function applyLang(lang){
  currentLang=lang;
  document.querySelectorAll('[data-i18n]').forEach(el=>el.textContent=t(el.dataset.i18n));
  const c=document.getElementById('nodeCount');
  if(c) document.getElementById('statusBackends').innerHTML=`${t('nodesLabel')}: <span id="nodeCount">${c.textContent}</span> ${t('online')}`;
  localStorage.setItem('rally_lang',lang);
}

function switchLang(lang){
  applyLang(lang);
  const a=document.querySelector('.tab.active');
  if(a){const t=a.dataset.tab;if(t==='dashboard')loadDashboard();if(t==='nodes')loadNodes();}
}

const API={
  async getConfig(){const r=await fetch('/api/config');if(!r.ok)throw new Error(await r.text());return r.json()},
  async saveConfig(c){const r=await fetch('/api/config',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(c)});if(!r.ok)throw new Error(await r.text());return r.json()},
  async getStatus(){const r=await fetch('/api/status');if(!r.ok)throw new Error(await r.text());return r.json()},
  async reload(){const r=await fetch('/api/reload',{method:'POST'});if(!r.ok)throw new Error(await r.text());return r.json()},
  async getRawConfig(){const r=await fetch('/api/config/raw');if(!r.ok)throw new Error(await r.text());return r.text()},
  async saveRawConfig(y){const r=await fetch('/api/config/raw',{method:'PUT',headers:{'Content-Type':'text/plain'},body:y});if(!r.ok)throw new Error(await r.text());return r.json()},
  async getStats(){const r=await fetch('/api/stats');if(!r.ok)throw new Error(await r.text());return r.json()},
  async getLogs(){const r=await fetch('/api/logs');if(!r.ok)throw new Error(await r.text());return r.json()},
  async toggleNode(name,enabled){const r=await fetch('/api/node/toggle',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name,enabled})});if(!r.ok)throw new Error(await r.text());return r.json()},
};

let configCache=null,logAutoScroll=true,logStreamActive=false;

function toast(msg,type='success'){
  const el=document.getElementById('toast');
  el.textContent=msg;el.className=`toast ${type} show`;
  clearTimeout(el._timer);
  el._timer=setTimeout(()=>el.classList.remove('show'),3000);
}

document.querySelectorAll('.tab').forEach(tab=>{
  tab.addEventListener('click',()=>{
    document.querySelectorAll('.tab').forEach(t=>t.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(c=>c.classList.remove('active'));
    tab.classList.add('active');
    document.getElementById('tab-'+tab.dataset.tab).classList.add('active');
    if(tab.dataset.tab==='dashboard')loadDashboard();
    if(tab.dataset.tab==='nodes')loadNodes();
    if(tab.dataset.tab==='logs')loadLogs();
    if(tab.dataset.tab==='config')loadConfigEditor();
  });
});

function onTypeChange(){
  const t=document.getElementById('nodeType').value;
  document.getElementById('rowPassword').style.display=(t==='vless')?'none':'';
  document.getElementById('rowSNI').style.display=(t==='hysteria2'||t==='trojan'||t==='vless')?'':'none';
  document.getElementById('rowCipher').style.display=(t==='ss')?'':'none';
  document.getElementById('rowUUID').style.display=(t==='vless')?'':'none';
  document.getElementById('rowFlow').style.display=(t==='vless')?'':'none';
  document.getElementById('rowDownMbps').style.display=(t==='hysteria2')?'':'none';
  document.getElementById('rowUpMbps').style.display=(t==='hysteria2')?'':'none';
  document.getElementById('nodePassword').required=(t!=='vless');
  document.getElementById('nodeUUID').required=(t==='vless');
}

function showNodeForm(index){
  const isEdit=index!==undefined;
  document.getElementById('editIndex').value=isEdit?index:'';
  document.getElementById('modalTitle').textContent=isEdit?t('editNodeTitle'):t('addNodeTitle');
  document.getElementById('btnSubmitNode').textContent=isEdit?t('editNodeBtn'):t('addNodeBtn');
  document.getElementById('nodeForm').reset();
  document.getElementById('nodeType').value='hysteria2';onTypeChange();
  if(isEdit&&configCache&&configCache.vps){
    const n=configCache.vps[index];
    document.getElementById('nodeName').value=n.name||'';
    document.getElementById('nodeType').value=n.type||'hysteria2';
    document.getElementById('nodeServer').value=n.server||'';
    document.getElementById('nodePort').value=n.port||'';
    document.getElementById('nodePassword').value=n.password||'';
    document.getElementById('nodeSNI').value=n.sni||'';
    document.getElementById('nodeCipher').value=n.cipher||'AEAD_CHACHA20_POLY1305';
    document.getElementById('nodeUUID').value=n.uuid||'';
    document.getElementById('nodeFlow').value=n.flow||'';
    document.getElementById('nodeDownMbps').value=n.down_mbps||'';
    document.getElementById('nodeUpMbps').value=n.up_mbps||'';onTypeChange();
  }
  document.getElementById('nodeModal').style.display='flex';
}

function closeNodeForm(){document.getElementById('nodeModal').style.display='none'}

async function submitNode(e){
  e.preventDefault();
  const t=document.getElementById('nodeType').value;
  const node={name:document.getElementById('nodeName').value,type:t,server:document.getElementById('nodeServer').value,port:parseInt(document.getElementById('nodePort').value)};
  if(t!=='vless')node.password=document.getElementById('nodePassword').value;
  if(t==='hysteria2'||t==='trojan'||t==='vless')node.sni=document.getElementById('nodeSNI').value||undefined;
  if(t==='ss')node.cipher=document.getElementById('nodeCipher').value;
  if(t==='vless'){node.uuid=document.getElementById('nodeUUID').value;node.flow=document.getElementById('nodeFlow').value||undefined}
  if(t==='hysteria2'){const dm=document.getElementById('nodeDownMbps').value,um=document.getElementById('nodeUpMbps').value;if(dm)node.down_mbps=parseInt(dm);if(um)node.up_mbps=parseInt(um)}
  try{
    const cfg=await API.getConfig(),idx=document.getElementById('editIndex').value;
    if(idx!=='')cfg.vps[parseInt(idx)]=node;else{cfg.vps=cfg.vps||[];cfg.vps.push(node)}
    await API.saveConfig(cfg);configCache=cfg;
    toast(idx!==''?t('nodeUpdated'):t('nodeAdded'));closeNodeForm();loadNodes();
  }catch(err){toast(err.message,'error')}
}

async function deleteNode(index){
  if(!confirm(t('deleteConfirm')))return;
  try{const cfg=await API.getConfig();cfg.vps.splice(index,1);await API.saveConfig(cfg);configCache=cfg;toast(t('nodeDeleted'));loadNodes()}catch(err){toast(err.message,'error')}
}

document.getElementById('nodeModal').addEventListener('click',e=>{if(e.target===e.currentTarget)closeNodeForm()});

// ─── Dashboard ───────────────────────────────────────────────────────────────

async function toggleNode(name,currentlyEnabled){
  try{
    await API.toggleNode(name,!currentlyEnabled);
    toast(t('nodeToggled'));
    loadDashboard();
  }catch(err){toast(err.message,'error')}
}

async function loadDashboard(){
  try{
    const status=await API.getStatus();
    const total=status.backends?status.backends.length:0;
    let active=0,disabled=0;
    status.backends.forEach(b=>{if(!b.enabled)disabled++;else if(b.connected)active++});
    document.getElementById('statTotal').textContent=total;
    document.getElementById('statActive').textContent=active;
    document.getElementById('statDisabled').textContent=disabled;
    document.getElementById('nodeCount').textContent=active;
    const nc=document.getElementById('nodeCount');
    if(nc)document.getElementById('statusBackends').innerHTML=`${t('nodesLabel')}: <span id="nodeCount">${nc.textContent}</span> ${t('online')}`;
    const pe=document.getElementById('statusProxy');
    pe.innerHTML=active>0?`● Proxy: <span class="online">${t('running')}</span>`:`● Proxy: <span class="offline">${t('stopped')}</span>`;
    const tb=document.querySelector('#dashboardTable tbody');tb.innerHTML='';
    status.backends.forEach(b=>{
      const tr=document.createElement('tr');
      const en=b.enabled!==false;
      tr.innerHTML=`
        <td>${esc(b.name)}</td>
        <td><span class="tag tag-${esc(b.type||'unknown')}">${esc(b.type||'-')}</span></td>
        <td>${esc(b.server||'-')}</td>
        <td><span class="tag ${en&&b.connected?'tag-online':'tag-offline'}">${en&&b.connected?t('onlineStatus'):t('offlineStatus')}</span></td>
        <td>${b.active||0}</td>
        <td style="color:var(--accent);font-family:monospace;font-size:12px" id="rate-${esc(b.name)}-down">-</td>
        <td style="color:var(--green);font-family:monospace;font-size:12px" id="rate-${esc(b.name)}-up">-</td>
        <td><label class="switch"><input type="checkbox" ${en?'checked':''} onchange="toggleNode('${esc(b.name)}', this.checked)"><span class="slider"></span></label></td>
      `;
      tb.appendChild(tr);
    });
    // Fetch traffic stats
    try{
      const st=await API.getStats();
      let dr=0,wr=0,dt=0,wt=0;
      st.forEach(s=>{
        dr+=s.read_bps||0;wr+=s.write_bps||0;dt+=s.read_total||0;wt+=s.write_total||0;
        const dn=document.getElementById('rate-'+s.name+'-down');
        const up=document.getElementById('rate-'+s.name+'-up');
        if(dn)dn.textContent=formatBps(s.read_bps||0);
        if(up)up.textContent=formatBps(s.write_bps||0);
      });
      const e1=document.getElementById("statDownRate");if(e1)e1.textContent=formatBps(dr);
      const e2=document.getElementById("statUpRate");if(e2)e2.textContent=formatBps(wr);
      const e3=document.getElementById("statDownTotal");if(e3)e3.textContent=formatBytes(dt);
      const e4=document.getElementById("statUpTotal");if(e4)e4.textContent=formatBytes(wt);
    }catch(_){}
  }catch(err){toast(t('failedLoadDashboard')+err.message,'error')}
}

// ─── Nodes ───────────────────────────────────────────────────────────────────

async function loadNodes(){
  try{
    configCache=await API.getConfig();
    const vps=configCache.vps||[],tb=document.querySelector('#nodeTable tbody');tb.innerHTML='';
    vps.forEach((n,i)=>{
      const tr=document.createElement('tr');
      let pw=n.password||"";const pwDisplay=pw?pw.slice(0,1)+"••••"+pw.slice(-1):"";
      tr.innerHTML=`<td><strong>${esc(n.name)}</strong></td><td><span class="tag tag-${esc(n.type||'hysteria2')}">${esc(n.type||'hysteria2')}</span></td><td>${esc(n.server)}</td><td>${n.port}</td><td style="font-family:monospace;font-size:12px;color:var(--text2)">${esc(pwDisplay)}</td><td><button class="btn-icon" onclick="showNodeForm(${i})">${t('edit')}</button><button class="btn-icon btn-danger" onclick="deleteNode(${i})">${t('delete')}</button></td>`;
      tb.appendChild(tr);
    });
  }catch(err){toast(t('failedLoadNodes')+err.message,'error')}
}

// ─── Logs ────────────────────────────────────────────────────────────────────

async function loadLogs(){
  try{
    const logs=await API.getLogs(),c=document.getElementById('logContent');c.innerHTML='';
    logs.forEach(e=>appendLogEntry(c,e));
    if(logAutoScroll){document.getElementById('logContainer').scrollTop=document.getElementById('logContainer').scrollHeight}
    if(!logStreamActive){logStreamActive=true;startLogStream()}
  }catch(err){toast(t('failedLoadDashboard')+err.message,'error')}
}

function appendLogEntry(c,e){
  const d=document.createElement('div');d.className='log-entry';
  d.innerHTML=`<span class="log-time">${esc(e.time)}</span><span class="log-level log-level-${esc(e.level)}">${esc(e.level)}</span><span class="log-msg">${esc(e.message)}</span>`;
  c.appendChild(d);
}

function startLogStream(){
  const es=new EventSource('/api/logs?mode=stream'),c=document.getElementById('logContent');
  es.onmessage=e=>{try{const entry=JSON.parse(e.data);appendLogEntry(c,entry);if(logAutoScroll){document.getElementById('logContainer').scrollTop=document.getElementById('logContainer').scrollHeight}}catch(_){}};
  es.onerror=()=>{es.close();logStreamActive=false;setTimeout(()=>{if(document.getElementById('tab-logs').classList.contains('active'))loadLogs()},2000)};
}

function clearLogs(){document.getElementById('logContent').innerHTML=''}

function toggleLogStream(){logAutoScroll=!logAutoScroll;document.getElementById('btnLogStream').style.opacity=logAutoScroll?'1':'0.5';if(logAutoScroll){document.getElementById('logContainer').scrollTop=document.getElementById('logContainer').scrollHeight}}

// ─── Config ─────────────────────────────────────────────────────────────────

async function loadConfigEditor(){try{document.getElementById('configEditor').value=await API.getRawConfig()}catch(err){toast(t('failedLoadConfig')+err.message,'error')}}
async function saveConfig(){try{await API.saveRawConfig(document.getElementById('configEditor').value);toast(t('configSaved'))}catch(err){toast(err.message,'error')}}
async function reloadConfig(){try{await API.reload();toast(t('configReloaded')+' — '+t('todo'));loadDashboard()}catch(err){toast(err.message,'error')}}

function formatBps(bps){if(bps<1024)return bps.toFixed(0)+' B/s';if(bps<1048576)return(bps/1024).toFixed(1)+' KB/s';return(bps/1048576).toFixed(2)+' MB/s';}
function formatBytes(b){if(b<1024)return b+' B';if(b<1048576)return(b/1024).toFixed(1)+' KB';if(b<1073741824)return(b/1048576).toFixed(1)+' MB';return(b/1073741824).toFixed(2)+' GB';}

function esc(s){if(s==null)return'';const d=document.createElement('div');d.textContent=String(s);return d.innerHTML}

// ─── Init ────────────────────────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded',()=>{
  document.getElementById('langSwitch').value=currentLang;
  applyLang(currentLang);loadDashboard();
  setInterval(()=>{if(document.getElementById('tab-dashboard').classList.contains('active'))loadDashboard()},10000);
});
