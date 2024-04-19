import OneSignal from 'react-native-onesignal'
import { emitter } from '../emitter'

interface AdditionalData {
  feedId: string
}
// https://documentation.onesignal.com/docs/react-native-sdk-setup
export const oneSignalInit = () => {
  /**
   * One signal setup
   */
  //OneSignal Init Code
  OneSignal.setLogLevel(6, 0)
  OneSignal.setAppId('fb0cfb9c-c9c0-4c05-a004-b052acfbc463')
  //END OneSignal Init Code

  //Prompt for push on iOS
  OneSignal.promptForPushNotificationsWithUserResponse((response) => {
    console.log('Prompt response:', response)
  })

  //Method for handling notifications received while app in foreground
  OneSignal.setNotificationWillShowInForegroundHandler(
    (notificationReceivedEvent) => {
      console.log(
        'OneSignal: notification will show in foreground:',
        notificationReceivedEvent,
      )
      const notification = notificationReceivedEvent.getNotification()
      console.log('notification: ', notification)
      const data = notification.additionalData
      console.log('additionalData: ', data)
      // Complete with null means don't show a notification.
      notificationReceivedEvent.complete(notification)
    },
  )

  //Method for handling notifications opened
  OneSignal.setNotificationOpenedHandler((openedEvent) => {
    console.log('OneSignal: notification opened:', openedEvent)
    const { action, notification } = openedEvent

    const { feedId } = notification.additionalData as AdditionalData
    if (feedId) {
      // if it doesn't work, then need to dispatch an action to set focusedColumnId
      // in redux store, currently focusedColumnId state is in ColumnFocusContext.tsx
      emitter.emit('FOCUS_ON_COLUMN', {
        animated: true,
        columnId: feedId,
        highlight: true,
        scrollTo: true,
      })
    }
  })
}

export { OneSignal }
