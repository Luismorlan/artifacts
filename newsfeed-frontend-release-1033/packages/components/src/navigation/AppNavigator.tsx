import React from 'react'

import {
  Router,
  Switch,
  Route,
  Redirect,
  RouteProps,
} from '../libs/react-router'
import { useReduxState } from '../hooks/use-redux-state'
import * as selectors from '../redux/selectors'
import { LoginScreen } from '../screens/LoginScreen'
import { MainScreen } from '../screens/MainScreen'
import { SharedPostScreen } from '../screens/SharedPostScreen'
import { AddSourceOrSubsourceScreen } from '../screens/AddSourceOrSubsourceScreen'
import { OneSignal } from '../libs/oneSignal'

export const RouteConfiguration = {
  root: '/',
  login: '/login',
  sharedPost: '/shared-posts/:id',
  addCustomizedSubsource: '/add-subsource',
}

export interface RouteParamsSharedPost {
  id: string
}

export const AppNavigator = React.memo(() => {
  const user = useReduxState(selectors.currentUserSelector)
  const userId = user?.id

  const PrivateRoute = ({ children, ...rest }: RouteProps) => {
    return (
      <Route
        {...rest}
        render={({ location }) =>
          user ? (
            children
          ) : (
            <Redirect
              to={{
                pathname: RouteConfiguration.login,
                state: {
                  from: location,
                },
              }}
            />
          )
        }
      />
    )
  }

  React.useEffect(() => {
    if (userId) {
      OneSignal.setExternalUserId(userId)
    }
  }, [userId])

  return (
    <Router>
      <Switch>
        <PrivateRoute exact path={RouteConfiguration.root}>
          <MainScreen key="app-main-screen" />
        </PrivateRoute>
        <Route exact path={RouteConfiguration.login}>
          <LoginScreen key="app-login-screen" />
        </Route>
        <Route exact path={RouteConfiguration.sharedPost}>
          <SharedPostScreen key="shared-post-screen" />
        </Route>
        <Route exact path={RouteConfiguration.addCustomizedSubsource}>
          <AddSourceOrSubsourceScreen
            isAddingSource={false}
            key="add-subsource-screen"
          />
        </Route>
      </Switch>
    </Router>
  )
})

AppNavigator.displayName = 'AppNavigator'
