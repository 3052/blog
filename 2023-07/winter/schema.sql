create table artist_t (
   artist_n integer primary key,
   artist_s text,
   check_s text,
   mb_s text
);

create table album_t (
   album_n integer primary key,
   album_s text,
   date_s text,
   url_s text
);

create table song_t (
   song_n integer primary key,
   song_s text,
   note_s text,
   album_n integer
);

create table song_artist_t (
   song_n integer,
   artist_n integer
);
