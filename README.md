# GoCentral
A Rock Band 3 master server re-implementation written in Golang using MongoDB as the database layer and my Quazal Rendez-vous-compatible forks of [nex-go](https://github.com/ihatecompvir/nex-go)/[nex-protocols-go](https://github.com/ihatecompvir/nex-protocols-go) as the underlying server layer. 

Note that this only aims to replicate what the game calls "Rock Central", support for the Music Store is _not_ here and will never be added. Just buy the songs through the Xbox Live Marketplace or PlayStation Store instead.

## Platform Compatibility
- PS3 (real hardware and RPCS3)
- Wii (real hardware and Dolphin)
- Xbox 360 (real hardware, requires RB3Enhanced)

## Setup and Usage
### Connecting on PS3 (Real Hardware)
1. Set your console's DNS settings to primary 45.33.44.103, secondary 1.1.1.1.
### Connecting on PS3 (RPCS3)
1. Ensure you have RPCN set up in RPCS3 and an account on RPCN.
2. In Settings->Network, make sure status is "Connected" and PSN status is "RPCN".
3. In "IP/Hosts switches", add `rb3ps3live.hmxservices.com=45.33.44.103`
### Connecting on Wii (RB3Enhanced)
1. Make sure RB3Enhanced 0.7 or later is installed. https://rb3e.rbenhanced.rocks/
2. GoCentral is enabled by default! If you need to enable it yourself:
    * Open rb3.ini, change GoCentralAddress to `gocentral-wii.rbenhanced.rocks`
### Connecting on Wii (Gecko/Ocarina Code) - Dolphin too!
1. Download the code from https://rb3e.rbenhanced.rocks/gocentral_gecko.txt
2. Copy this code to wherever you store Gecko/Ocarina codes on your SD card. This is often in txtcodes/SZBx69.txt, where x is your region (P for Europe, E for America).

(If on Dolphin, right click the game's properties and enter the code into the Gecko Codes tab.)
### Connecting on Wii (USB Loader GX)
1. Set your console's DNS settings to primary 45.33.44.103, secondary 1.1.1.1.
2. In your loader settings for Rock Band 3, enable the "NoSSL only" option for custom servers.
If you are using another loader, check with your loader on enabling a NoSSL patch. Will add more instructions later on.
### Connecting on Xbox 360 (RB3Enhanced)
1. Make sure RB3Enhanced 0.7 or later is installed. https://rb3e.rbenhanced.rocks/
2. GoCentral is enabled by default! If you need to enable it yourself:
    * Open rb3.ini, change GoCentralAddress to `gocentral-xbox.rbenhanced.rocks`

For the most reliable experience, port forward port 9103 (UDP) to your console in your router's settings, or if on RPCS3, enable UPnP.

(Do note that by changing DNS settings, you may be unable to play other games or use other services. Some ISPs may block custom DNS servers.)

## Features Implemented
- Message of the Day
- Online Matchmaking
- Leaderboards
- Entity storage (characters, bands)
- Linked account spoofing to unlock the "Link Your Account to Rockband.com" goal/achievement
- Battle of the Bands
- Setlist Challenges
- Setlist Sharing
- Global rank calculation
- Instaranks ("You are ranked #4 on the Guitar Leaderboard" on the post-song stats screen)

## Features Coming In the Future
- [Crossplay between PS3 and Wii](https://www.youtube.com/watch?v=KW5NrjDsv00) (requires RB3Enhanced)

## Special Thanks
The following users made contributions to GoCentral, but aren't listed in the Contributors tab on GitHub, so they are listed here instead.
- [@knvtva](https://github.com/knvtva)
- [@li1lypad](https://github.com/li1lypad)
