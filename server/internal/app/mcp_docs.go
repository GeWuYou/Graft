package app

import (
	"encoding/json"
	"fmt"

	productmcp "graft/server/internal/mcp"
)

const (
	mcpDocsPath     = "/mcp/docs"
	mcpDocsJSONPath = "/mcp/docs.json"
)

// buildMCPDocsCatalog 将 compiler 生成的 MCP catalog 与运行时连接信息组合成 Explorer 的唯一数据源。
func buildMCPDocsCatalog(bundle []byte, enabled bool) ([]byte, error) {
	catalog, err := productmcp.CompileDocumentationCatalog(bundle)
	if err != nil {
		return nil, err
	}
	catalogJSON, err := json.Marshal(catalog)
	if err != nil {
		return nil, fmt.Errorf("encode MCP documentation catalog: %w", err)
	}

	payload := struct {
		MCP struct {
			Endpoint       string `json:"endpoint"`
			Enabled        bool   `json:"enabled"`
			Authentication string `json:"authentication"`
		} `json:"mcp"`
		Catalog json.RawMessage `json:"catalog"`
	}{Catalog: catalogJSON}
	payload.MCP.Endpoint = "/mcp"
	payload.MCP.Enabled = enabled
	payload.MCP.Authentication = "Personal access token (PAT); token scopes are constrained by existing REST and RBAC authorization."

	result, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode MCP Explorer payload: %w", err)
	}
	return result, nil
}

func renderMCPDocsHTML() []byte {
	return []byte(mcpDocsPage)
}

const mcpDocsPage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="icon" type="image/svg+xml" href="/favicon.svg?v=3"><title>Graft MCP Explorer</title>
<style>
:root{color-scheme:light dark;font-family:Inter,ui-sans-serif,system-ui,sans-serif;color:#172033;background:#f7f8fa}*{box-sizing:border-box}body{margin:0}.shell{max-width:1280px;margin:auto;padding:32px 24px 56px}.top{display:flex;justify-content:space-between;align-items:start;gap:24px;border-bottom:1px solid #d9dce3;padding-bottom:24px}h1{margin:0;font-size:28px;letter-spacing:0}h2{font-size:16px;margin:0}p{color:#596174;line-height:1.5}.endpoint{font:13px ui-monospace,monospace;background:#e9edf3;padding:7px 9px;border-radius:4px}.toolbar{display:flex;gap:12px;align-items:center;margin:24px 0}.search{width:min(440px,100%);padding:10px 12px;border:1px solid #bbc1ce;border-radius:4px;font:inherit;background:Canvas;color:CanvasText}.filters{display:flex;gap:6px;flex-wrap:wrap}button{border:1px solid #bbc1ce;border-radius:4px;padding:8px 10px;background:Canvas;color:CanvasText;font:inherit;cursor:pointer}button[aria-pressed="true"]{background:#1b5ec9;color:white;border-color:#1b5ec9}.overview{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin:20px 0 24px}.stat{padding:16px;border:1px solid #d9dce3;border-radius:6px;background:Canvas}.stat strong{font-size:24px;display:block}.content{display:grid;grid-template-columns:minmax(250px,360px) minmax(0,1fr);gap:16px}.list,.details{border:1px solid #d9dce3;border-radius:6px;background:Canvas;min-height:300px}.list{padding:8px}.item{width:100%;text-align:left;border:0;border-bottom:1px solid #edf0f4;border-radius:0;padding:12px}.item:last-child{border:0}.item small{display:block;color:#596174;margin-top:4px}.details{padding:24px}.details pre{white-space:pre-wrap;overflow:auto;background:#f2f4f7;color:#172033;padding:12px;border-radius:4px}.label{font-size:12px;color:#596174;text-transform:uppercase;letter-spacing:.05em;margin:18px 0 6px}.status{font-size:13px;color:#596174}@media (prefers-color-scheme:dark){:root{color:#e7ebf2;background:#111827}.endpoint{background:#263247}.stat,.list,.details{background:#182233;border-color:#344157}.item{border-color:#263247}.details pre{background:#101827;color:#e7ebf2}p,.status,.item small,.label{color:#aeb8c8}}@media(max-width:720px){.shell{padding:20px 16px}.top,.toolbar{align-items:stretch;flex-direction:column}.overview,.content{grid-template-columns:1fr}.search{width:100%}}
</style></head><body><main class="shell"><header class="top"><div><h1>Graft MCP Explorer</h1><p>Capabilities compiled from the canonical OpenAPI contract.</p></div><code class="endpoint" id="endpoint">/mcp</code></header><section class="overview" aria-label="Overview"><div class="stat"><strong id="tools-count">-</strong>Tools</div><div class="stat"><strong id="resources-count">-</strong>Resources</div><div class="stat"><strong id="actions-count">-</strong>Actions</div></section><div class="toolbar"><input class="search" id="search" type="search" placeholder="Search capabilities" aria-label="Search capabilities"><div class="filters" aria-label="Filter capabilities"><button data-kind="all" aria-pressed="true">All</button><button data-kind="tools" aria-pressed="false">Tools</button><button data-kind="resources" aria-pressed="false">Resources</button><button data-kind="actions" aria-pressed="false">Actions</button></div></div><p class="status" id="status">Loading catalog...</p><section class="content"><div class="list" id="list" aria-label="Capabilities"></div><article class="details" id="details"><p>Select a capability to inspect its contract.</p></article></section></main><script>
const state={kind:'all',query:'',catalog:{tools:[],resources:[],actions:[]}};const $=id=>document.getElementById(id);const escape=value=>String(value??'').replace(/[&<>'"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]));function entries(){return ['tools','resources','actions'].flatMap(kind=>(state.catalog[kind]||[]).map(item=>({...item,_kind:kind}))).filter(item=>(state.kind==='all'||state.kind===item._kind)&&JSON.stringify(item).toLowerCase().includes(state.query.toLowerCase()))}function title(item){return item.name||item.operation_id||item.uri_template||'Unnamed capability'}function renderDetail(item){if(!item){$('details').innerHTML='<p>No matching capability.</p>';return}const schema=item.input_schema||item.inputSchema;const metadata=item.risk||item.confirmation?JSON.stringify({risk:item.risk,confirmation:item.confirmation},null,2):'';$('details').innerHTML='<h2>'+escape(title(item))+'</h2><p>'+escape(item.description||'No description provided.')+'</p><div class="label">Protocol mapping</div><code>'+escape([item.method,item.path||item.uri_template].filter(Boolean).join(' '))+'</code>'+(schema?'<div class="label">Input schema</div><pre>'+escape(JSON.stringify(schema,null,2))+'</pre>':'')+(metadata?'<div class="label">Safety</div><pre>'+escape(metadata)+'</pre>':'')}function render(){const found=entries();$('list').innerHTML=found.map((item,index)=>'<button class="item" data-index="'+index+'"><strong>'+escape(title(item))+'</strong><small>'+escape(item._kind.slice(0,-1)+' · '+[item.method,item.path||item.uri_template].filter(Boolean).join(' '))+'</small></button>').join('')||'<p>No matching capabilities.</p>';$('list').querySelectorAll('[data-index]').forEach(button=>button.addEventListener('click',()=>renderDetail(found[Number(button.dataset.index)])));renderDetail(found[0])}document.querySelectorAll('[data-kind]').forEach(button=>button.addEventListener('click',()=>{state.kind=button.dataset.kind;document.querySelectorAll('[data-kind]').forEach(other=>other.setAttribute('aria-pressed',String(other===button)));render()}));$('search').addEventListener('input',event=>{state.query=event.target.value;render()});fetch('/mcp/docs.json').then(response=>{if(!response.ok)throw Error('Unable to load catalog');return response.json()}).then(payload=>{state.catalog=payload.catalog||payload;const mcp=payload.mcp||{};$('endpoint').textContent=mcp.endpoint||'/mcp';['tools','resources','actions'].forEach(kind=>$(kind+'-count').textContent=(state.catalog[kind]||[]).length);$('status').textContent=mcp.enabled===false?'MCP runtime is disabled; the catalog remains available for contract inspection.':'Catalog loaded from the canonical OpenAPI contract.';render()}).catch(error=>{$('status').textContent=error.message;$('details').innerHTML='<p>The catalog could not be loaded.</p>'});
</script></body></html>`
