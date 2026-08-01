#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

extern void jameclawMenuNewChat(void);
extern void jameclawMenuAutomations(void);
extern void jameclawMenuShowDesktop(void);
extern void jameclawMenuShowConsole(void);

static NSString *const JameClawHomeNavigationNotification = @"com.jameclaw.home.navigate";

static void requestHomeSection(NSString *section, BOOL startNewChat) {
    // The native Jame window is a separate app process. Launch/activate it
    // first, then send the navigation request after its SwiftUI scene has had
    // time to become ready. Sending twice also covers a cold launch.
    jameclawMenuShowDesktop();
    for (NSNumber *delay in @[@0.7, @1.5]) {
        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(delay.doubleValue * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
            [[NSDistributedNotificationCenter defaultCenter]
                postNotificationName:JameClawHomeNavigationNotification
                object:@"com.jameclaw.launcher"
                userInfo:@{ @"section": section, @"new_chat": @(startNewChat) }
                options:NSNotificationDeliverImmediately];
        });
    }
}

@interface JameClawDesktopMenuTarget : NSObject
@end

@implementation JameClawDesktopMenuTarget
- (void)newChat:(id)sender { requestHomeSection(@"chat", YES); }
- (void)automations:(id)sender { requestHomeSection(@"automations", NO); }
- (void)showDesktop:(id)sender { jameclawMenuShowDesktop(); }
- (void)showConsole:(id)sender { jameclawMenuShowConsole(); }
- (void)applicationDidBecomeActive:(NSNotification *)notification { jameclawMenuShowDesktop(); }
@end

static JameClawDesktopMenuTarget *menuTarget;
static IMP originalApplicationShouldHandleReopen;

static BOOL jameclawApplicationShouldHandleReopen(id self, SEL selector, NSApplication *application, BOOL hasVisibleWindows) {
    // A Dock click on an already-active, windowless launcher is delivered as
    // a reopen request rather than another didBecomeActive notification.
    // Always raise the native Jame window, then preserve any launcher delegate
    // behavior that was already installed by the tray framework.
    jameclawMenuShowDesktop();
    if (originalApplicationShouldHandleReopen != NULL) {
        BOOL (*original)(id, SEL, NSApplication *, BOOL) = (void *)originalApplicationShouldHandleReopen;
        return original(self, selector, application, hasVisibleWindows);
    }
    return YES;
}

static void installDockReopenHandler(void) {
    id delegate = NSApp.delegate;
    if (delegate == nil) {
        return;
    }
    Class delegateClass = object_getClass(delegate);
    SEL selector = @selector(applicationShouldHandleReopen:hasVisibleWindows:);
    Method method = class_getInstanceMethod(delegateClass, selector);
    const char *types = "c@:@c";
    if (method != NULL) {
        originalApplicationShouldHandleReopen = method_getImplementation(method);
        // Add an override when the implementation is inherited so another
        // application's delegate class is never modified globally.
        if (class_addMethod(delegateClass, selector, (IMP)jameclawApplicationShouldHandleReopen, method_getTypeEncoding(method))) {
            return;
        }
        method = class_getInstanceMethod(delegateClass, selector);
        method_setImplementation(method, (IMP)jameclawApplicationShouldHandleReopen);
    } else {
        class_addMethod(delegateClass, selector, (IMP)jameclawApplicationShouldHandleReopen, types);
    }
}

static void addMenu(NSMenu *mainMenu, NSString *title, NSString *itemTitle, SEL action) {
    NSMenu *submenu = [[NSMenu alloc] initWithTitle:title];
    NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:itemTitle action:action keyEquivalent:@""];
    [item setTarget:menuTarget];
    [submenu addItem:item];

    NSMenuItem *topLevel = [[NSMenuItem alloc] initWithTitle:title action:nil keyEquivalent:@""];
    [topLevel setSubmenu:submenu];
    [topLevel setTag:9470 + [mainMenu numberOfItems]];
    [mainMenu addItem:topLevel];
}

void jameclawInstallDesktopMenu(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSMenu *mainMenu = NSApp.mainMenu;
        if (mainMenu == nil) {
            mainMenu = [[NSMenu alloc] initWithTitle:@"Main Menu"];
            [NSApp setMainMenu:mainMenu];
        }

        // The launcher can recreate its tray while recovering from an error;
        // only add this set of top-level menus once.
        if ([mainMenu itemWithTitle:@"New Chat"] != nil) {
            return;
        }

        menuTarget = [[JameClawDesktopMenuTarget alloc] init];
        // The launcher itself is intentionally windowless. Treat a Dock click
        // as a request to reveal its native Jame window, just like a regular
        // desktop app would reveal its main window.
        [[NSNotificationCenter defaultCenter] addObserver:menuTarget
                                                 selector:@selector(applicationDidBecomeActive:)
                                                     name:NSApplicationDidBecomeActiveNotification
                                                   object:NSApp];
        installDockReopenHandler();
        addMenu(mainMenu, @"New Chat", @"Start New Chat", @selector(newChat:));
        addMenu(mainMenu, @"Automations", @"Open Automations", @selector(automations:));

        NSMenu *viewMenu = [[NSMenu alloc] initWithTitle:@"View"];
        NSMenuItem *desktop = [[NSMenuItem alloc] initWithTitle:@"Show JameClaw Desktop" action:@selector(showDesktop:) keyEquivalent:@""];
        [desktop setTarget:menuTarget];
        [viewMenu addItem:desktop];
        NSMenuItem *console = [[NSMenuItem alloc] initWithTitle:@"Open Web Console" action:@selector(showConsole:) keyEquivalent:@""];
        [console setTarget:menuTarget];
        [viewMenu addItem:console];
        NSMenuItem *viewTopLevel = [[NSMenuItem alloc] initWithTitle:@"View" action:nil keyEquivalent:@""];
        [viewTopLevel setSubmenu:viewMenu];
        [mainMenu addItem:viewTopLevel];
    });
}
