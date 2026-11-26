var launch = document.getElementById('launch');
var cartridgeName = document.body.dataset.cartridge;
var cartridgeBytes;
var Module = window.Module = {
 canvas: document.getElementById('canvas'),
 arguments: ['--skip', '--cmd', 'load __play_' + cartridgeName + ' & run'],
 locateFile: function(name){ return new URL(name,document.baseURI).href; },
 printErr: function(message){ console.error('[PixelForge]',message); },
 onAbort: function(){ launch.hidden=false;launch.textContent='Could not start. Reload to retry.'; },
 onRuntimeInitialized: function(){
  var start=Module._sokol_start;
  Module._sokol_start=function(){
   var directory='/com.pixelforge/PixelForge';
   FS.mkdirTree(directory);
   FS.writeFile(directory+'/__play_'+cartridgeName+'.pf',cartridgeBytes);
   launch.hidden=true;
   return start.apply(this,arguments);
  };
 }
};
launch.addEventListener('click',async function(){
 launch.disabled=true;launch.textContent='Loading game…';
 try {
  var response=await fetch('cartridges/'+cartridgeName+'.pf');
  if(!response.ok)throw new Error('Cartridge download failed');
  cartridgeBytes=new Uint8Array(await response.arrayBuffer());
  var loader=document.createElement('script');loader.src='pixelforge.js?v=20261126';
  loader.onerror=function(){launch.disabled=false;launch.textContent='Connection interrupted. Reload to retry.';};
  document.body.appendChild(loader);Module.canvas.focus();
 } catch(error){console.error(error);launch.disabled=false;launch.textContent='Download failed. Reload to retry.';}
});
