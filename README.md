I don't know really much to say, but I built this because of Discord's upload size decrease to 10MB. A BetterDiscord & Vencord plugin coming soon.

![FileRipple](https://github.com/user-attachments/assets/bdb9d760-5c22-4dd3-9994-2082d2d0b903) 

/static/ Directory required for Logo & Favicon
/uploadedfiles/ Directory require for the uploaded files

API:
/api/v1/upload/ POST (CONTENT: file(FILE, POST) , user(COOKIE) )
/api/v1/download/user/file GET
/api/v1/delete/user/file GET (CONTENT: user(COOKIE) )
