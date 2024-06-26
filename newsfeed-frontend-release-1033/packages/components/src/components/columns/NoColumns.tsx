import React from 'react'

import { sharedStyles } from '../../styles/shared'
import { GenericMessageWithButtonView } from '../cards/GenericMessageWithButtonView'
import { Column, ColumnProps } from './Column'

export type NoColumnsProps = Omit<ColumnProps, 'columnId'>

export function NoColumns(props: NoColumnsProps) {
  return (
    <Column
      columnId=""
      {...props}
      style={[sharedStyles.center, sharedStyles.fullWidth, props.style]}
    >
      <GenericMessageWithButtonView
        buttonView={undefined}
        emoji="wave"
        title="Welcome to NewsFeed :)"
        subtitle={'Add a column by tapping the "+" button'}
      />
    </Column>
  )
}
