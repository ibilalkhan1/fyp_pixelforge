'use strict';
const status=document.getElementById('status'),restart=document.getElementById('restart'),sound=document.getElementById('sound'),frame=document.getElementById('gameframe');
setInterval(()=>{
 const game=frame.contentWindow;if(!game?.pfExampleState)return;
 const state=JSON.parse(game.pfExampleState());
 status.textContent=`${state.game} · Score ${state.score}`+(state.lives!==undefined?` · Lives ${state.lives}`:'')+` · Position ${state.x}, ${state.y}`+(state.gameOver?' · Game over / level ended — Restart to play again':' · Playing');
 for(const key of ['score','x','y','gameOver','lives'])status.dataset[key]=String(state[key]);
 sound.textContent=state.sound?'Sound: on':'Sound: off';restart.disabled=false;sound.disabled=false;
},150);
restart.onclick=()=>frame.contentWindow.pfExampleControl?.('restart');sound.onclick=()=>frame.contentWindow.pfExampleControl?.('sound');
function sendKey(type,key){frame.contentDocument.querySelector('canvas')?.dispatchEvent(new KeyboardEvent(type,{key,code:key,bubbles:true}))}
for(const button of document.querySelectorAll('[data-key]')){
 const key=button.dataset.key;
 button.onpointerdown=event=>{event.preventDefault();button.setPointerCapture(event.pointerId);sendKey('keydown',key)};
 for(const name of ['pointerup','pointercancel','lostpointercapture'])button.addEventListener(name,()=>sendKey('keyup',key));
}
window.addEventListener('keydown',event=>{if(event.key.startsWith('Arrow')){event.preventDefault();sendKey('keydown',event.key)}});
window.addEventListener('keyup',event=>{if(event.key.startsWith('Arrow')){event.preventDefault();sendKey('keyup',event.key)}});
