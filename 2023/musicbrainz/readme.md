# MusicBrainz

If you add a new release, the release group is created at the same time. Be sure
to include:

- release title
- artist
- type
- status
- date
- country
- label
- release link
- format
- track titles
- track lengths

<https://wiki.musicbrainz.org/MusicBrainz_API>

## Release length

The minimum is 179.5 seconds, which rounds up to 180 seconds, which is 3
minutes. A pathological example is here:

~~~xml
<track id="b9346c0a-1166-30e7-aba3-997ef3065abd">
   <position>4</position>
   <number>4</number>
   <length>179600</length>
   <recording id="0393ce29-889d-4e9a-930e-c110bb87626d">
      <title>In Our Angelhood</title>
      <length>179600</length>
   </recording>
</track>
~~~

<https://musicbrainz.org/ws/2/release/fed8322a-e8d7-4c65-867b-1697f6204395?inc=recordings>

measured by the millisecond, this track is too short. Measured by the second,
this track is long enough. Listed here at 2:59:

<https://youtube.com/watch?v=JVx0li_Hihk>

Listed here at 3:00:

<https://musicbrainz.org/release/fed8322a-e8d7-4c65-867b-1697f6204395>

Listed here at 3:01:

<https://youtube.com/watch?v=tNk-mlgXRp4>

To resolve this, we need to round to the second before making any decisions.
