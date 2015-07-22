package godis





var prefix="godis::"

func Consumers() string {   //set
	return  prefix + "comsumers"
}


func Producers() string {    //set
	return  prefix + "producers"
}


func ProducerService(service string) string {  //set  value:producer
	return  prefix +  "service::" + service
}


func ProducerMsgQueen(producer string) string {   //list
	return  prefix +  producer + "::message"
}




func ConsumerMsgQueen(consumer string) string {  //list
	return  prefix + consumer + "::message"
}

