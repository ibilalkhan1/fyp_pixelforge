'use strict';
document.getElementById('start').onclick=async()=>{
 const button=document.getElementById('start'),status=document.getElementById('loading');button.disabled=true;status.textContent='Loading the Go engine…';
 try {
  const response=await fetch(document.body.dataset.wasm || 'game.wasm');if(!response.ok)throw Error('Game download failed');
  const go=new Go();go.exit=code=>{if(code!==0){status.hidden=false;status.textContent='Game could not start (exit '+code+').';button.hidden=false;button.disabled=false}};const result=await WebAssembly.instantiate(await response.arrayBuffer(),go.importObject);
  button.hidden=true;status.hidden=true;go.run(result.instance).catch(error=>{status.hidden=false;status.textContent='Game stopped: '+error.message});
 } catch(error){status.textContent=error.message;button.disabled=false}
};
