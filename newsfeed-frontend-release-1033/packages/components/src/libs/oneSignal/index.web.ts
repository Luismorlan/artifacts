// todo: setup onesignal web client
// https://documentation.onesignal.com/docs/react-native-sdk-setup
export const oneSignalInit = () => {
  console.log('oneSignalInit called, do nothing for web')
}
const OneSignal = {
  setExternalUserId: (userId: string) => {
    console.log('setExternalUserId called, do nothing for web')
  },
}
export { OneSignal }
