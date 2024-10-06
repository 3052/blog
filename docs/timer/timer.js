var normalBackColor = "palegreen";
var normalForeColor = "navy";
var warnBackColor = "yellow";
var warnForeColor = "navy";
var alarmBackColor = "red";
var alarmForeColor = "white";
var doneBlink = false;
var showNegative = false;
var resetTime = 300;
var timeLeft = 300;
var warnTime = 60;
var alarmTime = 30;
var paused = true;
var roundTo = 10;
var warnRoundTo = 5;
var alarmRoundTo = 1;
var warnBeeps = 1; // Number of warning beeps to make
var alarmBeeps = 2; // Number of alarm beeps to make
var doneBeeps = 3; // Number of "done" beeps to make
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
    if (paused) return;
    var now = new Date();
    now = now.getTime();
    var left = target - now;
    if (left > 0 || doneBlink || showNegative) {
        var nextUpdate = left % 1000;
        if (nextUpdate == 0) nextUpdate = 1;
        else if (nextUpdate < 0) nextUpdate = 1000;
        pendingUpdate = setTimeout("update()", nextUpdate);
    }
}

function display() {
   var now = new Date();
   now = now.getTime();
   var left = target - now;
   if (paused) left = timeLeft; // Round to nearest second, being clever about negative numbers
      if ((left % 1000 + 1000) % 1000 < 500){
         left = Math.floor(left / 1000);
      } else{
         left = Math.ceil(left / 1000);
      }
   var blinkSeconds = left;
   if (left < 0 && !showNegative){
      left = 0; // Round up to multiple of n
   }
   var round = roundTo;
   if (left <= 0 && !doneBeepDone) {
      doneBeepDone = true;
      if (doneBeeps > 0){
         multibeep(doneBeeps, doneBeepGap, doneBeepDuration, doneBeepFreq);
      }
   }
   if (left <= alarmTime) {
      round = alarmRoundTo;
      if (!alarmBeepDone) {
         alarmBeepDone = true;
         if (alarmBeeps > 0){
            multibeep(alarmBeeps, alarmBeepGap, alarmBeepDuration, alarmBeepFreq);
         }
      }
   } else if (left <= warnTime) {
      round = warnRoundTo;
      if (!warnBeepDone) {
         warnBeepDone = true;
         if (warnBeeps > 0){
            multibeep(warnBeeps, warnBeepGap, warnBeepDuration, warnBeepFreq);
         }
      }
   }
   var rounded = Math.floor((left + round - 1) / round) * round;
   var minutes = Math.floor(rounded / 60);
   var seconds = rounded % 60;
   if (seconds < 0) {
      minutes += 1;
      seconds = -seconds;
      if (minutes == 0){
         minutes = "-" + minutes;
      }
   }
   var sec = seconds;
   if (seconds < 10){
      sec = "0" + seconds;
   }
   document.getElementById("countdown").innerHTML = minutes + ":" + sec;
   if (left <= 0 && doneBlink) { // Blink every two seconds.  Blinking is done by swapping the
      // foreground and background colors.
      if (-blinkSeconds % 2 == 1) {
         document.body.style.color = alarmForeColor;
         document.body.style.backgroundColor = alarmBackColor;
      } else {
         document.body.style.color = alarmBackColor;
         document.body.style.backgroundColor = alarmForeColor;
      }
   } else if (left <= alarmTime) {
      document.body.style.color = alarmForeColor;
      document.body.style.backgroundColor = alarmBackColor;
   } else if (left <= warnTime) {
      document.body.style.color = warnForeColor;
      document.body.style.backgroundColor = warnBackColor;
   } else {
      document.body.style.color = normalForeColor;
      document.body.style.backgroundColor = normalBackColor;
   }
}

// Start or stop the timer
function pause() {
    var now = new Date();
    now = now.getTime();
    var button = document.getElementById("pause-start-button");
    if (paused) {
        paused = false;
        target = now + timeLeft;
        clearTimeout(pendingUpdate);
        button.innerHTML = 'Pause';
        update();
    } else {
        paused = true;
        timeLeft = target - now;
        button.innerHTML = 'Start';
        display();
    }
}

// We need an audio context to be able to beep.  However, we can't do it
// here because some browsers insist that the user first take some action
// (to prevent annoying autoplay crap).  So we'll create it inside the
// beep function, one time only.
var audioCtx; // Beep function.  All arguments are optional.

//      duration of the tone in milliseconds. Default is 200.
//      frequency of the tone in hertz. default is 440.
//      volume of the tone. Default is 1, off is 0.
//      type of tone. Possible values are sine, square, sawtooth, triangle,
//        and custom. Default is sine.
//      callback to use at the end of the tone
function beep(duration, frequency, volume, type, callback) {
    if (!audioCtx) audioCtx = new(window.AudioContext || window.webkitAudioContext || window.audioContext);
    var oscillator = audioCtx.createOscillator();
    var gainNode = audioCtx.createGain();
    oscillator.connect(gainNode);
    gainNode.connect(audioCtx.destination);
    if (volume) gainNode.gain.value = volume;
    if (frequency) oscillator.frequency.value = frequency;
    if (type) oscillator.type = type;
    if (callback) oscillator.onended = callback;
    oscillator.start();
    setTimeout(function() {
        oscillator.stop();
    }, (duration ? duration : 200));
}

// Function that can beep several times.  Arguments are as for beep except:
//      count is number of beeps.  Default is 2.
//      gap is gap between beeps in ms.  Default is 300.
function multibeep(count, gap, duration, frequency, volume, type, callback) {
   if (!gap || gap <= 0){
      gap = 300;
   }
   if (!count || count <= 0){
      count = 2;
   }
   if (count == 1){
      cb = callback;
   } else {
      rebeep = function() {
      multibeep(count - 1, gap, duration, frequency, volume, type, callback);
      };
      cb = function() {
      setTimeout(rebeep, gap);
      };
   }
   beep(duration, frequency, volume, type, cb);
}

// Set up a dictionary that has all our default variables
var dictionary = [];
dictionary['normalBackColor'] = normalBackColor;
dictionary['normalForeColor'] = normalForeColor;
dictionary['warnBackColor'] = warnBackColor;
dictionary['warnForeColor'] = warnForeColor;
dictionary['alarmBackColor'] = alarmBackColor;
dictionary['alarmForeColor'] = alarmForeColor;

if (doneBlink){
   dictionary['doneBlink'] = "true";
} else{
   dictionary['doneBlink'] = "false";
}

if (showNegative){
   dictionary['showNegative'] = "true";
} else {
   dictionary['showNegative'] = "false";
}

dictionary['resetTimeMinutes'] = Math.floor(resetTime / 60);
dictionary['resetTimeSeconds'] = resetTime % 60;
dictionary['warnTime'] = warnTime;
dictionary['alarmTime'] = alarmTime;
dictionary['roundTo'] = roundTo;
dictionary['warnRoundTo'] = warnRoundTo;
dictionary['alarmRoundTo'] = alarmRoundTo;
dictionary['warnBeeps'] = warnBeeps;
dictionary['warnBeepDuration'] = warnBeepDuration;
dictionary['warnBeepFreq'] = warnBeepFreq;
dictionary['alarmBeeps'] = alarmBeeps;
dictionary['alarmBeepDuration'] = alarmBeepDuration;
dictionary['alarmBeepFreq'] = alarmBeepFreq;
dictionary['alarmBeepGap'] = alarmBeepGap;
dictionary['doneBeeps'] = doneBeeps;
dictionary['doneBeepDuration'] = doneBeepDuration;
dictionary['doneBeepFreq'] = doneBeepFreq;
dictionary['doneBeepGap'] = doneBeepGap;
dictionary['addMinutes'] = addMinutes;
dictionary['addSeconds'] = addSeconds; //

// Parse name/value pairs from the URL.
//
// First, strip off the leading '?'
var searchString = document.location.search;
searchString = searchString.substring(1);
var nvPairs = searchString.split("&");

// Now loop through the pairs, and extract what we want
for (i = 0; i < nvPairs.length; i++) {
    var nvPair = nvPairs[i].split("=");
    var name = nvPair[0];
    var value = decodeURIComponent(nvPair[1]);
    dictionary[name] = value;
}

// Pick out all variable values that we allow to be controlled from
// the URL
normalBackColor = dictionary['normalBackColor'];
normalForeColor = dictionary['normalForeColor'];
warnBackColor = dictionary['warnBackColor'];
warnForeColor = dictionary['warnForeColor'];
alarmBackColor = dictionary['alarmBackColor'];
alarmForeColor = dictionary['alarmForeColor'];

if (dictionary['doneBlink'] == "true"){
   doneBlink = true;
} else {
   doneBlink = false;
}

if (dictionary['showNegative'] == "true"){
   showNegative = true;
} else {
   showNegative = false;
}

resetTime = +dictionary['resetTimeMinutes'] * 60 + (+dictionary['resetTimeSeconds']);
warnTime = +dictionary['warnTime'];
alarmTime = +dictionary['alarmTime'];
roundTo = +dictionary['roundTo'];
warnRoundTo = +dictionary['warnRoundTo'];
alarmRoundTo = +dictionary['alarmRoundTo'];
warnBeeps = +dictionary['warnBeeps'];
warnBeepDuration = +dictionary['warnBeepDuration'];
warnBeepFreq = +dictionary['warnBeepFreq'];
warnBeepGap = +dictionary['warnBeepDuration'];
alarmBeeps = +dictionary['alarmBeeps'];
alarmBeepDuration = +dictionary['alarmBeepDuration'];
alarmBeepFreq = +dictionary['alarmBeepFreq'];
alarmBeepGap = +dictionary['alarmBeepDuration'];
doneBeeps = +dictionary['doneBeeps'];
doneBeepDuration = +dictionary['doneBeepDuration'];
doneBeepFreq = +dictionary['doneBeepFreq'];
doneBeepGap = +dictionary['doneBeepDuration'];
addMinutes = +dictionary['addMinutes'];
addSeconds = +dictionary['addSeconds'];
var now = new Date();
now = now.getTime();
var target = now + resetTime * 1000;
timeLeft = resetTime * 1000;

document.write(`
<center>
   <span id="countdown" style="font-size:50vw; font-family: sans-serif"></span>
</center>
`);

document.write("<center>");
document.write("<button id='pause-start-button' onclick='pause()'>Start</button>");
document.write("<br/><br/>");
document.write("</center>");
document.body.style.color = normalForeColor;
document.body.style.backgroundColor = normalBackColor;
display();
