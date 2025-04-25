# Audara Desktop App

> Currently Windows only. Yeah, I'm a Windows-only app dev. Yes, we exist. Yes, we're valid. No go make better linux drivers, nerd. J/K I'll make a linux and mac version one day.

## What is it?

I built Audara for one very specific purpose in mind : I wanted to have the ability to remote control part of my computer, via my cell phone. 

It sounds silly, but sometimes, I just throw some music on there, pump the speakers up, and go do some cleaning. If I don't like the song that's playing, *could* I walk back to the computer 
and change the song? Sure I could! But *can* I spend weeks building an app that can let me login to a website and control my device remotely directly in my palm? Abso-fucking-lutely!

So, I built a website, with a server, that supports websocket and Clerk auth. And then I built a Go app that could connect to it, through the same website, to match the login system.

So now, when I press play/pause on my phone... my computer pauses my media! Thousands upon thousands of lines of code, megabytes of RAM, CPU cycles for days... just to press a button 
remotely. Ain't technology lovely? 

## How it works

It's a simple 3 step process!

### Step 1: Download and install Audara App

> Hey I'm not done building this yet, so, like, imagine that you actually have a download link here. 
> Or you could just clone and run this repo if you want, tbh.

### Step 2: Connect the App to the Website

> This is where you'd open the app and click "Login" *If you had one!*

### Step 3: Login to the app from your mobile device

> I'd put up a website that says "Under construction" but I haven't figured out the hosting yet. Honestly I don't know why you're here reading this. Who *are* you?

## What it can't do yet

Uhhhh most things, I haven't finished building this yet. I got *some* things done but I am really far from an MVP. Just be patient ;)

But at MVP lauch, the plan is: 

In the App: 

- An app that runs in the Windows system Tray (should work on Windows 10 and 11), that lets you login through a browser
- This app stays running and connects to a websocket, waiting for instructions
- If it receives a command, it runs the appropriate keyboard commands, like Play/Pause, Next, Volume Up/Down, etc!

In the Web: 

- A website to log into through either Google or Facebook (more would require paying for Clerk, I'm not there yet!)
- A simple indicator telling you if your desktop is online 
- Simple media controls (play/pause, back, next, mute, volume up, volume down)
- Only one PC supported at launch, probably

## What it will do later on

- Service/autostart access (not sure how to do that yet)
- Support multiple desktop client (would this be a Pro version? Maybe!)
- Linux version (requires specific OS calls and list of virtual keycode, different build, etc)
- MacOS version (ditto, actually)
- Support guest access (invite someone, they can control the music)
- Potentially launch music app (youtube music? spotify? WINAMP?) remotely to start some music

## Want to contribute? 

Hey, I'm game! I want to do this on my own mostly at the moment, because ideas are worth shit if you've got no means of implementing them, and I want my idea to be worth something.

However, while I'm pretty good at web development, full stack frameworks are not my forte, so Tanstack Start and Clerk are new territories for me. As for Go... well I just don't even
know the language. How did I build this app, then, you might ask? Well, I vibe coded it... I'm not even kidding. As a senior web developer, having a pretty good grasp on general CS
concepts, I was able to coax Claude/Sonnet using [Cursor](https://www.cursor.com/) , and build this entire thing from the ground up basically without writing a single line of code. 
In fact, this README is many times more characters than all the code I've actually had to write manually in a file - but not nearly as much as the amount of prompting I've had to do. 

So, if you're a Go developer and this horrifies you, and you just want to dig in and prove that a *real* dev can do better than shitty AI, then... I won't stop you! I'll *hinder* 
you, maybe, by being slow with PR reviews and such, but that's just what it costs to work on a team. Hit me up, we'll chat about it!
