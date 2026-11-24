'use strict';
const status=document.getElementById('status'),restart=document.getElementById('restart');
const sound=document.getElementById('sound'),pause=document.getElementById('pause');
const frame=document.getElementById('gameframe'),objective=document.getElementById('objective');
const heldKeys=new Set();
setInterval(()=>{
 const game=frame.contentWindow;if(!game?.pfExampleState)return;
 const state=JSON.parse(game.pfExampleState());
 const ended=state.won===undefined?'Game over / level ended — Restart to play again':'Game over — Restart to try again';
 const phase=state.won?'Level complete!':state.gameOver?ended:state.paused?'Paused':'Playing';
 status.textContent=`${state.game} · Score ${state.score}`+(state.lives!==undefined?` · Lives ${state.lives}`:'')+` · Position ${state.x}, ${state.y} · ${phase}`;
 for(const key of ['score','x','y','gameOver','lives','won','paused','driving','deliveries','hasKey'])status.dataset[key]=String(state[key]);
 if(sound){sound.textContent=state.sound?'Sound: on':'Sound: off';sound.disabled=false}
 if(pause){pause.textContent=state.paused?'Resume':'Pause';pause.disabled=!!(state.gameOver||state.won)}
 if(objective&&state.message)objective.textContent=state.message;
 restart.disabled=false;
},150);
restart.onclick=()=>{releaseKeys();frame.contentWindow.pfExampleControl?.('restart')};
if(sound)sound.onclick=()=>frame.contentWindow.pfExampleControl?.('sound');
if(pause)pause.onclick=()=>{releaseKeys();frame.contentWindow.pfExampleControl?.('pause')};
function sendKey(type,code){
 const key=code==='Space'?' ':code.startsWith('Key')?code.slice(3).toLowerCase():code;
 frame.contentDocument.querySelector('canvas')?.dispatchEvent(new KeyboardEvent(type,{key,code,bubbles:true}));
 if(type==='keydown')heldKeys.add(code);else heldKeys.delete(code);
}
function releaseKeys(){for(const key of [...heldKeys])sendKey('keyup',key)}
for(const button of document.querySelectorAll('[data-key]')){
 const key=button.dataset.key;
 button.onpointerdown=event=>{event.preventDefault();button.setPointerCapture(event.pointerId);sendKey('keydown',key)};
 for(const name of ['pointerup','pointercancel','lostpointercapture'])button.addEventListener(name,()=>sendKey('keyup',key));
}
const allowed=new Set(['ArrowUp','ArrowDown','ArrowLeft','ArrowRight','Space','KeyW','KeyA','KeyS','KeyD','KeyR','KeyP']);
window.addEventListener('keydown',event=>{if(allowed.has(event.code)){event.preventDefault();sendKey('keydown',event.code)}});
window.addEventListener('keyup',event=>{if(allowed.has(event.code)){event.preventDefault();sendKey('keyup',event.code)}});
window.addEventListener('blur',releaseKeys);
