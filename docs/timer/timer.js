'use strict';

document.write('<h1>1.0.1</h1>');

///

var doneBlink = false;

var resetTime = 300;
var timeLeft = 300;
var warnTime = 60
var alarmTime = 30

var paused = true;

var roundTo = 10;
var warnRoundTo = 5;
var alarmRoundTo = 1;

var warnBeeps = 1;              // Number of warning beeps to make
var alarmBeeps = 2;             // Number of alarm beeps to make
var doneBeeps = 3;              // Number of "done" beeps to make

var warnBeepDuration = 200;
var warnBeepFreq = 440;
var warnBeepGap = 300;
var alarmBeepDuration = 200;
var alarmBeepFreq = 440;
var alarmBeepGap = 200;
var doneBeepDuration = 200;
var doneBeepFreq = 440;
var doneBeepGap = 200;

var warnBeepDone = false;
var alarmBeepDone = false;
var doneBeepDone = false;

var addMinutes = 1;
var addSeconds = 0;

var pendingUpdate = 0;

function update() {
   display();
   if (paused){
      return;
   }
   var now = new Date();
   now = now.getTime();
   var left = target - now;
   var nextUpdate = left % 1000
   if (nextUpdate == 0){
      nextUpdate = 1
   } else if (nextUpdate < 0){
      nextUpdate = 1000
   }
   pendingUpdate = setTimeout("update()", nextUpdate)
}

function display() {
   var now = new Date();
   now = now.getTime();
   var left = target - now;
   if (paused){
      left = timeLeft
   }
   // Round to nearest second, being clever about negative numbers
   if ((left % 1000 + 1000) % 1000 < 500){
      left = Math.floor(left / 1000);
   } else{
      left = Math.ceil(left / 1000);
   }
   var blinkSeconds = left;
   if (left < 0  &&  !true){
      left = 0
   }
   // Round up to multiple of n
   var round = roundTo;
   if (left <= 0  &&  !doneBeepDone) {
      doneBeepDone = true;
      if (doneBeeps > 0){
         multibeep(doneBeeps, doneBeepGap, doneBeepDuration, doneBeepFreq);
      }
   }
   if (left <= alarmTime) {
      round = alarmRoundTo
      if (!alarmBeepDone) {
         alarmBeepDone = true
         if (alarmBeeps > 0){
            multibeep(alarmBeeps, alarmBeepGap, alarmBeepDuration, alarmBeepFreq)
         }
      }
   } else if (left <= warnTime) {
      round = warnRoundTo
      if (!warnBeepDone) {
         warnBeepDone = true
         if (warnBeeps > 0){
            multibeep(warnBeeps, warnBeepGap, warnBeepDuration, warnBeepFreq)
         }
      }
   }
   var rounded = Math.floor((left + round - 1) / round) * round;
   var minutes = Math.floor(rounded / 60);
   var seconds = rounded % 60;
   if (seconds < 0) {
      minutes += 1
      seconds = -seconds
      if (minutes == 0){
         minutes = "-" + minutes
      }
   }
   var sec = seconds;
   if (seconds < 10){
      sec = "0" + seconds;
   }
   document.getElementById("countdown").innerHTML = minutes + ":" + sec
}

// Start or stop the timer
function pause() {
   var now = new Date()
   now = now.getTime()
   var button = document.getElementById("pause-start-button")
   if (paused) {
      paused = false
      target = now + timeLeft
      clearTimeout(pendingUpdate)
      button.innerHTML = 'Pause'
      update()
   } else {
      paused = true
      timeLeft = target - now
      button.innerHTML = 'Start'
      display();
   }
}

// We need an audio context to be able to beep.  However, we can't do it
// here because some browsers insist that the user first take some action
// (to prevent annoying autoplay crap).  So we'll create it inside the
// beep function, one time only.
var audioCtx;

// Beep function.  All arguments are optional.
//      duration of the tone in milliseconds. Default is 200.
//      frequency of the tone in hertz. default is 440.
//      volume of the tone. Default is 1, off is 0.
//      type of tone. Possible values are sine, square, sawtooth, triangle,
//        and custom. Default is sine.
//      callback to use at the end of the tone
function beep(duration, frequency, volume, type, callback) {
    if (!audioCtx)
        audioCtx = new(window.AudioContext  ||  window.webkitAudioContext
          ||  window.audioContext);

    var oscillator = audioCtx.createOscillator()
    var gainNode = audioCtx.createGain()

    oscillator.connect(gainNode)
    gainNode.connect(audioCtx.destination)

    if (volume)
        gainNode.gain.value = volume
    if (frequency)
        oscillator.frequency.value = frequency
    if (type)
        oscillator.type = type
    if (callback)
        oscillator.onended = callback

    oscillator.start()
    setTimeout(function(){oscillator.stop()}, (duration ? duration : 200))
}

// Function that can beep several times.  Arguments are as for beep except:
//      count is number of beeps.  Default is 2.
//      gap is gap between beeps in ms.  Default is 300.
function multibeep(count, gap, duration, frequency, volume, type, callback) {
   if (!gap  ||  gap <= 0)
      gap = 300
   if (!count  ||  count <= 0) {
      count = 2
   }
   let cb;
   if (count == 1) {
      cb = callback;
   } else {
      const rebeep = function() {
         multibeep(count - 1, gap, duration, frequency, volume, type, callback)
      };
      cb = function() {
         setTimeout(rebeep, gap);
      };
   }
   beep(duration, frequency, volume, type, cb)
}

resetTime = 9;

var now = new Date();
now = now.getTime()
var target = now + resetTime * 1000
timeLeft = resetTime * 1000

document.write(`
<center>
   <span id="countdown"></span>
</center>
<center>
   <button id="pause-start-button" onclick="pause()">Start</button>
</center>
`);

display();
