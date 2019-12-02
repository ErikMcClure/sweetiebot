Sweetiebot is modular and can easily incorporate additional modules or commands. A command is a struct that satisfies the `Command` interface. 

    type Command interface {
      Name() string
      Process([]string, *discordgo.Message, []int, *GuildInfo) (string, bool, *discordgo.MessageEmbed)
      Usage(*GuildInfo) *CommandUsage
      UsageShort() string
    }
    
`Name()` returns the actual text that invokes the command, `Usage()` is a long, structured explanation of the command and it's parameters, and `UsageShort()` is a much shorter explanation of the command, both used by `!help`. `Process()` is called when Sweetiebot evaluates a command and matches it with this command's name (case-insensitive). The first `[]string` parameter is a list of the arguments to the command, which are seperated by spaces, unless they were surrounded by double-quotes `"`, just how command-line arguments work on all standard operating systems.

Commands belong to Modules, and are automatically added when adding a module. Modules are more complicated and respond to certain events in the chat if they are enabled. At minimum, a module must implement the `Module` interface:

    type Module interface {
      Name() string
      Commands() []Command
      Description() string
    }
    
`Name()` returns the name of the module, only used for enabling or restricting the module configuration. `Description()` is called by `!help` and should briefly describe the module's purpose. `Commands()` should return an initialized list of all commands associated with the module. The guild will automatically register the module for all hook interfaces that it satisfies. A module must satisfy the interface of the hook it is trying to add itself to, which simply means implementing a hook function with the appropriate parameters.
    
You can access the bot database using `info.Bot.DB`, but this will only work for server-independent database information (like users or transcripts), or on servers that have permission to write to the database. Additional modules will always be disabled on existing servers until they are explicitely enabled. [Submit a pull request](https://github.com/erikmcclure/sweetiebot/pull/new/master) if you'd like to contribute!

Before submitting a pull request, please make sure your code builds against the `master` branch of sweetiebot, while using the develop branch of `erikmcclure/discordgo`.