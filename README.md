# SigninBot
A sample signin bot in GoLang

This sample bot demonstrates how to use bot to help users sign-up and sign-in your website.

You could try the bot here https://t.me/kybersigninbot. Note that this sample doesn't actually sign you in, just give you the idea. In addition, actual implementation should save state to some database like [Bolt](https://github.com/boltdb/bolt).

The bot use Golang Telegram Wrapper [telebot](https://gopkg.in/tucnak/telebot.v2).

Recently, Telegram introduce the [Login Widget](https://core.telegram.org/widgets/login) so you might want to try it as an alternative approach for Telegram Sign-in.
