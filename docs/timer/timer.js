'use strict';

document.querySelector('h2').textContent = '1.1.3';

class Duration {
   // requires Android Firefox 79
   static millisecond = 1;
   static second = 1000 * Duration.millisecond;
   static minute = 60 * Duration.second;
   constructor(value) {
      this.value = value;
   }
   abs() {
      if (this.value >= 0) {
         return new Duration(this.value);
      }
      return new Duration(-this.value);
   }
   minutes() {
      return this.value / Duration.minute;
   }
   seconds() {
      return this.value / Duration.second;
   }
   string() {
      let s = '';
      if (this.value < 0) {
         s = '-';
      }
      let abs = this.abs();
      const minute = abs.truncate(Duration.minute).minutes();
      if (minute > 0) {
         s += minute.toString() + 'm ';
         abs.value -= minute * Duration.minute;
      }
      return s + abs.seconds() + 's';
   }
   truncate(value) {
      return new Duration(this.value - this.value % value);
   }
}

const param = new URLSearchParams(location.search);

const end = Date.now() +
   param.get('m') * Duration.minute +
   param.get('s') * Duration.second;

// We need an audio context to be able to beep.  However, we can't do it
// here because some browsers insist that the user first take some action
// (to prevent annoying autoplay crap).  So we'll create it inside the
// beep function, one time only.
let audio;

function interval() {
   const begin = Date.now();
   const until = new Duration(end-begin);
   if (until.value <= 0) {
      if (audio == undefined) {
         audio = new AudioContext;
      }
      const oscillator = audio.createOscillator();
      const gain = audio.createGain();
      oscillator.connect(gain);
      gain.connect(audio.destination);
      document.querySelector('h3').textContent = 'start';
      oscillator.start();
      setTimeout(function() {
         oscillator.stop();
      }, 99);
   }
   document.querySelector('h1').textContent = until.string();
}

setInterval(interval, 999);
